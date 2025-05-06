package sql

import (
	"context"
	"fmt"

	"github.com/fsdevblog/shorturl/internal/repositories"
	"github.com/jackc/pgx/v5"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type URLRepo struct {
	conn *pgxpool.Pool
}

func NewURLRepo(conn *pgxpool.Pool) *URLRepo {
	return &URLRepo{
		conn: conn,
	}
}

const batchCreateURLQuery = `-- batchCreateURLs
INSERT INTO urls 
	(short_identifier, url) 
VALUES ($1, $2)
ON CONFLICT (url, short_identifier) 
	DO UPDATE SET updated_at = NOW()
RETURNING id, created_at, updated_at, short_identifier, url, xmax = 0 AS inserted;
`

func (u *URLRepo) BatchCreate(
	ctx context.Context,
	args []repositories.BatchCreateArg,
) ([]repositories.BatchResult[models.URL], error) {
	batch := new(pgx.Batch)

	for _, arg := range args {
		vals := []interface{}{arg.ShortIdentifier, arg.URL}
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
			&inserted,
		)
		if err != nil {
			m.Err = ConvertErrorType(err)
		}
		if !inserted {
			m.Err = repositories.ErrDuplicateKey
		}
		ret[i] = m
	}
	if err := bResults.Close(); err != nil {
		return nil, fmt.Errorf("failed to close batch: %w", err)
	}
	return ret, nil
}

const createURLQuery = `-- createURL
INSERT INTO urls (short_identifier, url) VALUES ($1, $2) 
RETURNING id, created_at, updated_at, short_identifier, url;
`

func (u *URLRepo) Create(ctx context.Context, modelURL *models.URL) error {
	row := u.conn.QueryRow(ctx, createURLQuery, modelURL.ShortIdentifier, modelURL.URL)

	var m models.URL
	scanErr := row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt, &m.ShortIdentifier, &m.URL)
	if scanErr != nil {
		return ConvertErrorType(scanErr)
	}
	*modelURL = m
	return nil
}

const getByShortIdentifierQuery = `-- getByShortIdentifier
SELECT id, short_identifier, url FROM urls WHERE short_identifier = $1;
`

func (u *URLRepo) GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error) {
	row := u.conn.QueryRow(ctx, getByShortIdentifierQuery, shortID)
	var m models.URL
	scanErr := row.Scan(&m.ID, &m.ShortIdentifier, &m.URL)
	if scanErr != nil {
		return nil, ConvertErrorType(scanErr)
	}
	return &m, nil
}

const getByURLQuery = `-- getByURL
SELECT id, short_identifier, url FROM urls WHERE url = $1;
`

func (u *URLRepo) GetByURL(ctx context.Context, rawURL string) (*models.URL, error) {
	row := u.conn.QueryRow(ctx, getByURLQuery, rawURL)
	var m models.URL
	scanErr := row.Scan(&m.ID, &m.ShortIdentifier, &m.URL)
	if scanErr != nil {
		return nil, ConvertErrorType(scanErr)
	}
	return &m, nil
}

const getAllURLsQuery = `-- getAllURLs
SELECT id, short_identifier, url FROM urls;;
`

func (u *URLRepo) GetAll(ctx context.Context) ([]models.URL, error) {
	var urls []models.URL
	rows, qErr := u.conn.Query(ctx, getAllURLsQuery)
	if qErr != nil {
		return nil, ConvertErrorType(qErr)
	}
	defer rows.Close()
	for rows.Next() {
		var m models.URL
		if err := rows.Scan(&m.ID, &m.ShortIdentifier, &m.URL); err != nil {
			return nil, ConvertErrorType(err)
		}
		urls = append(urls, m)
	}
	if err := rows.Err(); err != nil {
		return nil, ConvertErrorType(err)
	}
	return urls, nil
}
