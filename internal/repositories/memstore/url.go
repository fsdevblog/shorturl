package memstore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/db/memory"
	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"
)

// URLRepo представляет собой репозиторий для работы с URL в памяти.
type URLRepo struct {
	s  *db.MemoryStorage
	mu sync.Mutex
}

// NewURLRepo создает новый экземпляр репозитория URL.
//
// Параметры:
//   - store: экземпляр хранилища в памяти
//
// Возвращает:
//   - *URLRepo: инициализированный репозиторий
func NewURLRepo(store *db.MemoryStorage) *URLRepo {
	return &URLRepo{
		s: store,
	}
}

// BatchCreate создает несколько URL записей одновременно.
//
// Параметры:
//   - ctx: контекст выполнения
//   - mURLs: слайс с данными для создания URL
//
// Возвращает:
//   - *repositories.BatchCreateShortURLsResult: результат создания записей
//   - error: ошибка выполнения операции (преобразованная через convertErrorType)
func (u *URLRepo) BatchCreate(
	ctx context.Context,
	mURLs []repositories.BatchCreateArg,
) (*repositories.BatchCreateShortURLsResult, error) {
	var collection = make(map[string]*models.URL, len(mURLs))
	for _, m := range mURLs {
		collection[m.ShortIdentifier] = &models.URL{
			URL:             m.URL,
			ShortIdentifier: m.ShortIdentifier,
			VisitorUUID:     m.VisitorUUID,
		}
	}
	br := memory.BatchSet[models.URL](ctx, collection, u.s.MStorage)

	var result = make([]repositories.BatchResult[models.URL], len(br))
	for i, r := range br {
		result[i] = repositories.BatchResult[models.URL]{
			Value: models.URL{
				URL:             collection[r.Key].URL,
				ShortIdentifier: collection[r.Key].ShortIdentifier,
				VisitorUUID:     collection[r.Key].VisitorUUID,
			},
			Err: convertErrorType(r.Err),
		}
	}

	return &repositories.BatchCreateShortURLsResult{
		Results: result,
	}, nil
}

// Create создает новую URL запись.
//
// Параметры:
//   - ctx: контекст выполнения
//   - sURL: данные URL для создания
//
// Возвращает:
//   - *models.URL: созданная запись
//   - bool: флаг успешного создания
//   - error: ошибка создания (преобразованная через convertErrorType)
func (u *URLRepo) Create(ctx context.Context, sURL *models.URL) (*models.URL, bool, error) {
	if err := memory.Set[models.URL](ctx, sURL.ShortIdentifier, sURL, u.s.MStorage); err != nil {
		if errors.Is(err, memory.ErrDuplicateKey) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf(
			"failed to create record: %w",
			convertErrorType(err),
		)
	}
	return sURL, true, nil
}

// GetByShortIdentifier получает URL по короткому идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения
//   - shortID: короткий идентификатор URL
//
// Возвращает:
//   - *models.URL: найденная запись
//   - error: ошибка поиска (преобразованная через convertErrorType)
func (u *URLRepo) GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error) {
	url, err := memory.Get[models.URL](ctx, shortID, u.s.MStorage)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get record by short identifier %s: %w",
			shortID, convertErrorType(err),
		)
	}
	return url, nil
}

// GetAllByVisitorUUID получает все URL для указанного посетителя.
//
// Параметры:
//   - ctx: контекст выполнения
//   - visitorUUID: идентификатор посетителя
//
// Возвращает:
//   - []models.URL: найденные записи
//   - error: ошибка поиска (преобразованная через convertErrorType)
func (u *URLRepo) GetAllByVisitorUUID(ctx context.Context, visitorUUID string) ([]models.URL, error) {
	data, err := memory.FilterAll[models.URL](ctx, u.s.MStorage, func(val models.URL) bool {
		if val.VisitorUUID == "" {
			return false
		}
		return val.VisitorUUID == visitorUUID
	})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get record by visitor uuid %s: %w",
			visitorUUID, convertErrorType(err),
		)
	}
	return data, nil
}

// GetByURL получает запись по оригинальному URL.
//
// Параметры:
//   - ctx: контекст выполнения
//   - rawURL: оригинальный URL
//
// Возвращает:
//   - *models.URL: найденная запись
//   - error: ошибка поиска (преобразованная через convertErrorType)
func (u *URLRepo) GetByURL(ctx context.Context, rawURL string) (*models.URL, error) {
	data, err := memory.GetAll[models.URL](ctx, u.s.MStorage)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get record by url %s records: %w",
			rawURL, convertErrorType(err),
		)
	}

	for _, val := range data {
		if val.URL == rawURL {
			return &val, nil
		}
	}
	return nil, repositories.ErrNotFound
}

// GetAll получает все сохраненные URL записи.
//
// Параметры:
//   - ctx: контекст выполнения
//
// Возвращает:
//   - []models.URL: все записи
//   - error: ошибка получения (преобразованная через convertErrorType)
func (u *URLRepo) GetAll(ctx context.Context) ([]models.URL, error) {
	urls, err := memory.GetAll[models.URL](ctx, u.s.MStorage)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get all records: %w",
			convertErrorType(err),
		)
	}
	return urls, nil
}

// DeleteByShortIDsVisitorUUID помечает URL записи как удаленные для указанного посетителя.
//
// Параметры:
//   - ctx: контекст выполнения
//   - visitorUUID: идентификатор посетителя
//   - shortIDs: список коротких идентификаторов для удаления
//
// Возвращает:
//   - error: ошибка удаления (преобразованная через convertErrorType)
func (u *URLRepo) DeleteByShortIDsVisitorUUID(ctx context.Context, visitorUUID string, shortIDs []string) (err error) { //nolint:nonamedreturns,lll
	u.mu.Lock()
	defer u.mu.Unlock()

	data, err := memory.FilterAll[models.URL](ctx, u.s.MStorage, func(val models.URL) bool {
		if val.VisitorUUID == "" {
			return false
		}
		return val.VisitorUUID == visitorUUID && slices.Contains(shortIDs, val.ShortIdentifier)
	})
	if err != nil {
		return convertErrorType(err)
	}
	now := time.Now().UTC()
	var batchMap = make(map[string]*models.URL, len(data))
	for i := range data {
		data[i].DeletedAt = &now
		batchMap[data[i].ShortIdentifier] = &data[i]
	}
	res := memory.BatchSet[models.URL](ctx, batchMap, u.s.MStorage, memory.WithOverwrite())
	for _, re := range res {
		if re.Err != nil {
			err = errors.Join(err, convertErrorType(re.Err))
		}
	}
	return err
}
