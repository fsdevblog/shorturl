package sql

import (
	"context"

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

type CreateURLParams struct {
	ShortIdentifier string
	URL             string
}

const createURLQuery = `-- createURL
INSERT INTO urls (short_identifier, url) VALUES ($1, $2) RETURNING *;
`

func (u *URLRepo) Create(ctx context.Context, modelURL *models.URL) error {
	row := u.conn.QueryRow(ctx, createURLQuery, modelURL.ShortIdentifier, modelURL.URL)

	var m models.URL
	scanErr := row.Scan(&m.ID, &m.ShortIdentifier, &m.URL)
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
	return &m, ConvertErrorType(scanErr)
}

const getByURLQuery = `-- getByURL
SELECT id, short_identifier, url FROM urls WHERE url = $1;
`

func (u *URLRepo) GetByURL(ctx context.Context, rawURL string) (*models.URL, error) {
	row := u.conn.QueryRow(ctx, getByURLQuery, rawURL)
	var m models.URL
	scanErr := row.Scan(&m.ID, &m.ShortIdentifier, &m.URL)
	return &m, ConvertErrorType(scanErr)
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
