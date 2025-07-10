package sql

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fsdevblog/shorturl/internal/repositories"
	"github.com/jackc/pgx/v5"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// URLRepo представляет собой репозиторий для работы с URL в PostgreSQL.
type URLRepo struct {
	conn *pgxpool.Pool
}

// NewURLRepo создает новый экземпляр репозитория URL.
//
// Параметры:
//   - conn: пул подключений к PostgreSQL
//
// Возвращает:
//   - *URLRepo: инициализированный репозиторий
func NewURLRepo(conn *pgxpool.Pool) *URLRepo {
	return &URLRepo{
		conn: conn,
	}
}

const batchCreateURLQuery = `-- batchCreateURLs
INSERT INTO urls 
	(short_identifier, url, visitor_uuid) 
VALUES ($1, $2, $3)
ON CONFLICT (url, visitor_uuid) 
	DO UPDATE SET updated_at = NOW()
RETURNING id, created_at, updated_at, short_identifier, url, visitor_uuid, xmax = 0 AS inserted;
`

// BatchCreate создает несколько URL записей одновременно.
//
// Параметры:
//   - ctx: контекст выполнения
//   - args: слайс с данными для создания URL
//
// Возвращает:
//   - *repositories.BatchCreateShortURLsResult: результат создания записей
//   - error: ошибка выполнения операции (преобразованная через convertErrType)
func (u *URLRepo) BatchCreate(
	ctx context.Context,
	args []repositories.BatchCreateArg,
) (*repositories.BatchCreateShortURLsResult, error) {
	batch := new(pgx.Batch)

	for _, arg := range args {
		vals := []interface{}{arg.ShortIdentifier, arg.URL, arg.VisitorUUID}
		batch.Queue(batchCreateURLQuery, vals...)
	}
	bResults := u.conn.SendBatch(ctx, batch)
	var ret = make([]repositories.BatchResult[models.URL], len(args))
	for i := range args {
		var inserted bool
		var m repositories.BatchResult[models.URL]
		err := bResults.QueryRow().Scan(
			&m.Value.ID,
			&m.Value.CreatedAt,
			&m.Value.UpdatedAt,
			&m.Value.ShortIdentifier,
			&m.Value.URL,
			&m.Value.VisitorUUID,
			&inserted,
		)
		if err != nil {
			m.Err = convertErrType(err)
		} else if !inserted && m.Value.ID != 0 {
			// Если запись получена но при этом не вставлена, нужно указать что она не уникальна.
			m.Err = repositories.ErrDuplicateKey
		}
		ret[i] = m
	}
	if err := bResults.Close(); err != nil {
		return nil, fmt.Errorf("failed to close batch: %w", err)
	}
	return &repositories.BatchCreateShortURLsResult{Results: ret}, nil
}

const createURLQuery = `-- createURL
INSERT INTO urls (short_identifier, url, visitor_uuid) 
	VALUES ($1, $2, $3) 
ON CONFLICT (url, visitor_uuid) 
	DO UPDATE SET updated_at = NOW()
RETURNING id, created_at, updated_at, short_identifier, url, visitor_uuid, xmax = 0 AS inserted;
`

// Create создает новую URL запись.
//
// Параметры:
//   - ctx: контекст выполнения
//   - modelURL: данные URL для создания
//
// Возвращает:
//   - *models.URL: созданная запись
//   - bool: флаг успешного создания (true если создана новая запись, false если обновлена существующая)
//   - error: ошибка создания (преобразованная через convertErrType)
func (u *URLRepo) Create(ctx context.Context, modelURL *models.URL) (*models.URL, bool, error) {
	row := u.conn.QueryRow(ctx, createURLQuery, modelURL.ShortIdentifier, modelURL.URL, modelURL.VisitorUUID)

	var m models.URL
	var inserted bool
	scanErr := row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt, &m.ShortIdentifier, &m.URL, &m.VisitorUUID, &inserted)
	if scanErr != nil {
		return nil, false, convertErrType(scanErr)
	}
	return &m, inserted, nil
}

const getByShortIdentifierQuery = `-- getByShortIdentifier
SELECT id, short_identifier, url, visitor_uuid, deleted_at FROM urls WHERE short_identifier = $1;
`

// GetByShortIdentifier получает URL по короткому идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения
//   - shortID: короткий идентификатор URL
//
// Возвращает:
//   - *models.URL: найденная запись
//   - error: ошибка поиска (преобразованная через convertErrType)
func (u *URLRepo) GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error) {
	row := u.conn.QueryRow(ctx, getByShortIdentifierQuery, shortID)
	var m models.URL
	scanErr := row.Scan(&m.ID, &m.ShortIdentifier, &m.URL, &m.VisitorUUID, &m.DeletedAt)
	if scanErr != nil {
		return nil, convertErrType(scanErr)
	}
	return &m, nil
}

const getAllByVisitorUUIDQuery = `-- getAllByVisitorUUID
SELECT id, short_identifier, url, visitor_uuid FROM urls WHERE visitor_uuid = $1;
`

// GetAllByVisitorUUID получает все URL для указанного посетителя.
//
// Параметры:
//   - ctx: контекст выполнения
//   - visitorUUID: идентификатор посетителя
//
// Возвращает:
//   - []models.URL: найденные записи
//   - error: ошибка поиска (преобразованная через convertErrType)
func (u *URLRepo) GetAllByVisitorUUID(ctx context.Context, visitorUUID string) ([]models.URL, error) {
	rows, qErr := u.conn.Query(ctx, getAllByVisitorUUIDQuery, visitorUUID)
	if qErr != nil {
		return nil, convertErrType(qErr)
	}
	defer rows.Close()

	var urls []models.URL
	for rows.Next() {
		var m models.URL
		if err := rows.Scan(&m.ID, &m.ShortIdentifier, &m.URL, &m.VisitorUUID); err != nil {
			return nil, convertErrType(err)
		}
		urls = append(urls, m)
	}

	if rErr := rows.Err(); rErr != nil {
		return nil, convertErrType(rErr)
	}

	return urls, nil
}

const getByURLQuery = `-- getByURL
SELECT id, short_identifier, url, visitor_uuid FROM urls WHERE url = $1;
`

// GetByURL получает запись по оригинальному URL.
//
// Параметры:
//   - ctx: контекст выполнения
//   - rawURL: оригинальный URL
//
// Возвращает:
//   - *models.URL: найденная запись
//   - error: ошибка поиска (преобразованная через convertErrType)
func (u *URLRepo) GetByURL(ctx context.Context, rawURL string) (*models.URL, error) {
	row := u.conn.QueryRow(ctx, getByURLQuery, rawURL)
	var m models.URL
	scanErr := row.Scan(&m.ID, &m.ShortIdentifier, &m.URL, &m.VisitorUUID)
	if scanErr != nil {
		return nil, convertErrType(scanErr)
	}
	return &m, nil
}

const getAllURLsQuery = `-- getAllURLs
SELECT id, short_identifier, url, visitor_uuid FROM urls;;
`

// GetAll получает все сохраненные URL записи.
//
// Параметры:
//   - ctx: контекст выполнения
//
// Возвращает:
//   - []models.URL: все записи
//   - error: ошибка получения (преобразованная через convertErrType)
func (u *URLRepo) GetAll(ctx context.Context) ([]models.URL, error) {
	var urls []models.URL
	rows, qErr := u.conn.Query(ctx, getAllURLsQuery)
	if qErr != nil {
		return nil, convertErrType(qErr)
	}
	defer rows.Close()
	for rows.Next() {
		var m models.URL
		if err := rows.Scan(&m.ID, &m.ShortIdentifier, &m.URL, &m.VisitorUUID); err != nil {
			return nil, convertErrType(err)
		}
		urls = append(urls, m)
	}
	if err := rows.Err(); err != nil {
		return nil, convertErrType(err)
	}
	return urls, nil
}

const markAsDeletedByShortIDVisitorUUIDQuery = `-- markAsDeletedByShortIDVisitorUUID
UPDATE urls SET deleted_at = NOW() WHERE short_identifier = $1 AND visitor_uuid = $2;
`

// DeleteByShortIDsVisitorUUID помечает URL записи как удаленные для указанного посетителя.
// Операция выполняется в транзакции батчами по 100 записей.
//
// Параметры:
//   - ctx: контекст выполнения
//   - visitorUUID: идентификатор посетителя
//   - shortIDs: список коротких идентификаторов для удаления
//
// Возвращает:
//   - error: ошибка удаления (преобразованная через convertErrType)
func (u *URLRepo) DeleteByShortIDsVisitorUUID(ctx context.Context, visitorUUID string, shortIDs []string) (err error) {
	tx, txErr := u.conn.Begin(ctx)
	if txErr != nil {
		return convertErrType(txErr)
	}

	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			if !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				err = convertErrType(rollbackErr)
			}
		}
	}()

	batchFlatInCh := make(chan error, len(shortIDs))

	const batchSize = 100

	lenShortIDs := len(shortIDs)
	wg := new(sync.WaitGroup)

	// разделяем запросы на батчи
	for i := 0; i < lenShortIDs; i += batchSize {
		end := i + batchSize
		if end > lenShortIDs {
			end = lenShortIDs
		}
		wg.Add(1)
		go markAsDeletedBatchFn(ctx, markAsDeletedBatchFnArgs{
			ids:         shortIDs[i:end],
			visitorUUID: visitorUUID,
			flatInCh:    batchFlatInCh,
			wg:          wg,
			tx:          tx,
		})
	}
	wg.Wait()

	close(batchFlatInCh)

	for batchErr := range batchFlatInCh {
		if batchErr != nil {
			if err != nil {
				err = errors.Join(err, convertErrType(batchErr))
			} else {
				err = convertErrType(batchErr)
			}
			return err
		}
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return convertErrType(fmt.Errorf("commit error: %w", commitErr))
	}

	return err
}

// markAsDeletedBatchFnArgs содержит аргументы для функции markAsDeletedBatchFn.
type markAsDeletedBatchFnArgs struct {
	ids         []string
	visitorUUID string
	flatInCh    chan error
	wg          *sync.WaitGroup
	tx          pgx.Tx
}

// markAsDeletedBatchFn обрабатывает батч запросов на удаление URL.
//
// Параметры:
//   - ctx: контекст выполнения
//   - args: аргументы для обработки батча
func markAsDeletedBatchFn(ctx context.Context, args markAsDeletedBatchFnArgs) {
	defer args.wg.Done()

	batch := new(pgx.Batch)
	for _, shortID := range args.ids {
		batch.Queue(markAsDeletedByShortIDVisitorUUIDQuery, shortID, args.visitorUUID)
	}

	bResults := args.tx.SendBatch(ctx, batch)

	for range args.ids {
		_, execErr := bResults.Exec()
		if execErr != nil {
			args.flatInCh <- convertErrType(execErr)
			return
		}
	}

	if closeErr := bResults.Close(); closeErr != nil {
		args.flatInCh <- convertErrType(closeErr)
		return
	}
	args.flatInCh <- nil
}
