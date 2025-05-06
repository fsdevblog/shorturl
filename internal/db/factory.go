package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StorageType string

const (
	StorageTypePostgres StorageType = "postgres"
	StorageTypeInMemory StorageType = "inMemory"
)

type FactoryConfig struct {
	StorageType  StorageType
	PostgresDSN  *string
	SqliteDBPath *string
}

func NewConnectionFactory(ctx context.Context, config FactoryConfig) (any, error) {
	switch config.StorageType {
	case StorageTypePostgres:
		if config.PostgresDSN == nil {
			return nil, errors.New("postgres dsn is empty")
		}
		pool, err := NewPostgresConnection(ctx, *config.PostgresDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres connection: %w", err)
		}
		// пока не будем ничего усложнять, а сделаем миграцию прямо здесь
		migrateErr := simpleMigrateSchema(ctx, pool)
		if migrateErr != nil {
			return nil, fmt.Errorf("failed to migrate schema: %w", migrateErr)
		}
		return pool, nil
	case StorageTypeInMemory:
		return NewMemStorage(), nil
	default:
		return nil, fmt.Errorf("unknown storage type: %s", config.StorageType)
	}
}

// да да, знаю что нужно миграции прикрутить людские). Обязательно сделаю.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    url VARCHAR(512) NOT NULL,
    short_identifier VARCHAR(8) NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_url_short_identifier ON urls (url, short_identifier);
`

func simpleMigrateSchema(ctx context.Context, conn *pgxpool.Pool) error {
	_, err := conn.Exec(ctx, schemaSQL)
	return err //nolint:wrapcheck
}
