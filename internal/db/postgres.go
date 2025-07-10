package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresConnection создает новый пул подключений к PostgreSQL.
//
// Параметры:
//   - ctx: контекст выполнения
//   - dsn: строка подключения к базе данных (Data Source Name)
//
// Возвращает:
//   - *pgxpool.Pool: пул подключений к PostgreSQL
//   - error: ошибка создания подключения
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
