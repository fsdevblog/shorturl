package services

import (
	"bufio"
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"

	"github.com/pkg/errors"
)

type URLRepository interface {
	BatchCreate(ctx context.Context, mURLs []repositories.BatchCreateArg) ([]repositories.BatchResult[models.URL], error)
	// Create вычисляет хеш короткой ссылки и создает запись в хранилище.
	Create(ctx context.Context, mURL *models.URL) error
	// GetByShortIdentifier находит в хранилище запись по заданному хешу ссылки
	GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error)
	// GetByURL находит запись в хранилище по заданной ссылке
	GetByURL(ctx context.Context, rawURL string) (*models.URL, error)
	// GetAll возвращает все записи в бд. Сразу пачкой.
	GetAll(ctx context.Context) ([]models.URL, error)
}

// URLService Сервис работает с базой данных в контексте таблицы `urls`.
type URLService struct {
	urlRepo URLRepository
}

func NewURLService(urlRepo URLRepository) *URLService {
	return &URLService{urlRepo: urlRepo}
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

type BatchExecResponse[T any] struct {
	Result T
	Err    error
}

func (u *URLService) BatchCreate(ctx context.Context, rawURLs []string) ([]models.URL, error) {
	// я проигнорирую здесь вопрос с коллизиями. В батч вставке обработка коллизий - это сущий кошмар который
	// врядли входит в рамки курса, а у меня мозг плавится, не успеваю к дедлайну по сдаче ревью)
	// из метода Create её также стоит убрать по идее.

	var args = make([]repositories.BatchCreateArg, len(rawURLs))
	delta := uint(1)
	for i, rawURL := range rawURLs {
		arg := repositories.BatchCreateArg{
			URL:             rawURL,
			ShortIdentifier: generateShortID(rawURL, delta, models.ShortIdentifierLength),
		}
		args[i] = arg
	}

	batchResults, batchErr := u.urlRepo.BatchCreate(ctx, args)
	if batchErr != nil {
		return nil, fmt.Errorf("%w: batch create: %s", ErrUnknown, batchErr.Error())
	}
	var m = make([]models.URL, len(batchResults))
	for i, result := range batchResults {
		m[i] = result.Value
		// if result.Err != nil {
		//	// тут потом разберусь..
		//}
	}
	return m, nil
}

func (u *URLService) Create(ctx context.Context, rawURL string) (*models.URL, error) {
	existingURL, existingURLErr := u.urlRepo.GetByURL(ctx, rawURL)
	if existingURLErr == nil {
		return existingURL, nil
	}

	var delta uint = 1
	var deltaMax uint = 10

	var sURL models.URL
	for {
		if delta >= deltaMax {
			return nil, fmt.Errorf("generateShortID loop limit for url: %w", ErrUnknown)
		}
		sURL = models.URL{
			URL:             rawURL,
			ShortIdentifier: generateShortID(rawURL, delta, models.ShortIdentifierLength),
		}
		if createErr := u.urlRepo.Create(ctx, &sURL); createErr != nil {
			if errors.Is(createErr, repositories.ErrDuplicateKey) {
				delta++
				continue
			}
			return nil, ErrUnknown
		}
		return &sURL, nil
	}
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

	// Читаем построчно. Batch вставку делать некогда если честно.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record models.URL
		if jsonErr := json.Unmarshal(scanner.Bytes(), &record); jsonErr != nil {
			return fmt.Errorf("unmarshal record: %w", jsonErr)
		}
		createErr := u.urlRepo.Create(ctx, &record)

		if createErr != nil {
			if !errors.Is(createErr, repositories.ErrDuplicateKey) {
				return fmt.Errorf("create record: %w", createErr)
			}
		}
	}
	return nil
}

// generateShortID генерирует идентификатор для ссылки нужной длины на основе delta.
func generateShortID(rawURL string, delta uint, length int) string {
	// Добавляем счетчик к срезу (для избежания коллизий)
	b := []byte(rawURL)
	b = append(b, byte(delta))

	// Создаем хеш и конвертим в base62
	hash := md5.Sum(b) //nolint:gosec
	base62 := base64.URLEncoding.EncodeToString(hash[:])
	return base62[:length]
}
