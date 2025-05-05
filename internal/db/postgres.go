package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresConnection(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	poolConfig, confErr := pgxpool.ParseConfig(dsn)
	if confErr != nil {
		return nil, fmt.Errorf("failed to parse config: %w", confErr)
	}
	pool, poolErr := pgxpool.NewWithConfig(ctx, poolConfig)
	if poolErr != nil {
		return nil, fmt.Errorf("failed to create pool: %w", poolErr)
	}
	return pool, nil
}
