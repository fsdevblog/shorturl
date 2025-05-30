package services

import (
	"bufio"
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"
)

type URLRepository interface {
	BatchCreate(ctx context.Context, mURLs []repositories.BatchCreateArg) ([]repositories.BatchResult[models.URL], error)
	// Create вычисляет хеш короткой ссылки и создает запись в хранилище.
	// Возвращает два значения: bool отвечает за уникальность созданной записи, 2 ошибку.
	Create(ctx context.Context, mURL *models.URL) (*models.URL, bool, error)
	// GetByShortIdentifier находит в хранилище запись по заданному хешу ссылки
	GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error)
	// GetByURL находит запись в хранилище по заданной ссылке
	GetByURL(ctx context.Context, rawURL string) (*models.URL, error)
	// GetAll возвращает все записи в бд. Сразу пачкой.
	GetAll(ctx context.Context) ([]models.URL, error)
	// GetAllByVisitorUUID возвращает записи связанные с visitorUUID.
	GetAllByVisitorUUID(ctx context.Context, visitorUUID string) ([]models.URL, error)
	// DeleteByShortIDsVisitorUUID помечает записи как удаленные.
	DeleteByShortIDsVisitorUUID(ctx context.Context, visitorUUID string, shortIDs []string) error
}

// URLService Сервис работает с базой данных в контексте таблицы `urls`.
type URLService struct {
	urlRepo URLRepository
}

func NewURLService(urlRepo URLRepository) *URLService {
	return &URLService{urlRepo: urlRepo}
}

func (u *URLService) GetAllByVisitorUUID(ctx context.Context, visitorUUID string) ([]models.URL, error) {
	urls, err := u.urlRepo.GetAllByVisitorUUID(ctx, visitorUUID)
	if err != nil {
		return nil, fmt.Errorf("get by visitor uuid: %w", err)
	}
	logrus.Infof("Get all urls by visitor uuid: %s: %+v", visitorUUID, urls)
	return urls, nil
}

func (u *URLService) GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error) {
	sURL, err := u.urlRepo.GetByShortIdentifier(ctx, shortID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, fmt.Errorf("id `%s` not found: %w", shortID, ErrRecordNotFound)
		}
		return nil, ErrUnknown
	}
	return sURL, nil
}

// BatchCreate Создает урлы пачками.
func (u *URLService) BatchCreate(
	ctx context.Context,
	visitorUUID string,
	rawURLs []string,
) (*BatchCreateShortURLsResponse, error) {
	var args = make([]repositories.BatchCreateArg, len(rawURLs))
	for i, rawURL := range rawURLs {
		arg := repositories.BatchCreateArg{
			URL:             rawURL,
			ShortIdentifier: generateShortID(rawURL, models.ShortIdentifierLength, visitorUUID),
			VisitorUUID:     visitorUUID,
		}
		args[i] = arg
	}

	batchResults, batchErr := u.urlRepo.BatchCreate(ctx, args)
	if batchErr != nil {
		return nil, fmt.Errorf("%w: batch create: %s", ErrUnknown, batchErr.Error())
	}
	batchResponse := NewBatchExecResponse[models.URL](len(batchResults))

	for i, result := range batchResults {
		batchResponse.results[i].Item = result.Value
		var err = result.Err
		if result.Err != nil && errors.Is(result.Err, repositories.ErrDuplicateKey) {
			err = ErrDuplicateKey
		}
		batchResponse.results[i].Err = err
	}
	return NewBatchExecResponseURL(batchResponse), nil
}

func (u *URLService) GetByURL(ctx context.Context, rawURL string) (*models.URL, error) {
	res, err := u.urlRepo.GetByURL(ctx, rawURL)

	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("%w: get by url: %s", ErrUnknown, err.Error())
	}
	return res, nil
}

func (u *URLService) Create(ctx context.Context, visitorUUID string, rawURL string) (*models.URL, bool, error) {
	var sURL = models.URL{
		URL:             rawURL,
		ShortIdentifier: generateShortID(rawURL, models.ShortIdentifierLength, visitorUUID),
		VisitorUUID:     visitorUUID,
	}
	m, isUniq, createErr := u.urlRepo.Create(ctx, &sURL)
	if createErr != nil {
		return nil, false, fmt.Errorf("%w: create: %s", ErrUnknown, createErr.Error())
	}
	return m, isUniq, nil
}

func (u *URLService) Backup(ctx context.Context, path string) (err error) {
	backupFile, backupFileErr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if backupFileErr != nil {
		return fmt.Errorf("open backup file: %w", backupFileErr)
	}

	defer func() {
		if errClose := backupFile.Close(); errClose != nil {
			err = fmt.Errorf("close backup file: %w", errClose)
		}
	}()

	records, recordsErr := u.urlRepo.GetAll(ctx)
	if recordsErr != nil {
		return fmt.Errorf("get all records for backup: %w", recordsErr)
	}
	for _, record := range records {
		j, e := json.Marshal(&record)
		if e != nil {
			return fmt.Errorf("marshal record %+v: %w", records, e)
		}
		j = append(j, '\n')
		_, wE := backupFile.Write(j)
		if wE != nil {
			return fmt.Errorf("write backup file: %w", wE)
		}
	}
	return nil
}

func (u *URLService) RestoreBackup(ctx context.Context, path string) (err error) {
	file, fileErr := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if fileErr != nil {
		return fmt.Errorf("open backup file: %w", fileErr)
	}
	defer func() {
		if errClose := file.Close(); errClose != nil {
			err = fmt.Errorf("close backup file: %w", errClose)
		}
	}()

	batchLimit := 1000
	batch := make([]repositories.BatchCreateArg, 0, batchLimit)
	// Читаем построчно. Batch вставку делать некогда если честно.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record models.URL
		if jsonErr := json.Unmarshal(scanner.Bytes(), &record); jsonErr != nil {
			return fmt.Errorf("unmarshal record: %w", jsonErr)
		}
		batch = append(batch, repositories.BatchCreateArg{
			ShortIdentifier: record.ShortIdentifier,
			URL:             record.URL,
		})

		if len(batch) == batchLimit {
			_, batchErr := u.urlRepo.BatchCreate(ctx, batch)
			if batchErr != nil {
				return fmt.Errorf("batch create: %w", batchErr)
			}
			batch = make([]repositories.BatchCreateArg, 0, batchLimit)
		}
	}
	if len(batch) > 0 {
		_, batchErr := u.urlRepo.BatchCreate(ctx, batch)
		if batchErr != nil {
			return fmt.Errorf("batch create: %w", batchErr)
		}
	}
	return nil
}

func (u *URLService) MarkAsDeleted(ctx context.Context, shortIDs []string, visitorUUID string) error {
	if err := u.urlRepo.DeleteByShortIDsVisitorUUID(ctx, visitorUUID, shortIDs); err != nil {
		return fmt.Errorf("delete by short ids %+v, visitor uuid: %s: %w", shortIDs, visitorUUID, err)
	}
	return nil
}

// generateShortID генерирует идентификатор для ссылки нужной длины на основе visitorUUID.
func generateShortID(rawURL string, length int, visitorUUID string) string {
	// Добавляем счетчик к срезу (для избежания коллизий)
	b := []byte(rawURL)
	b = append(b, []byte(visitorUUID)...)

	// Создаем хеш и конвертим в base62
	hash := md5.Sum(b) //nolint:gosec
	base62 := base64.URLEncoding.EncodeToString(hash[:])
	return base62[:length]
}
