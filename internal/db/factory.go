package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// driver for migration applying postgres.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// driver to get migrations from files (*.sql in our case).
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// StorageType определяет тип хранилища данных.
type StorageType string

// StorageTypePostgres PostgreSQL.
// StorageTypeInMemory In-memory хранилище.
// PostgresDefaultMigrationsDir путь к директории с миграциями по умолчанию.
const (
	StorageTypePostgres          StorageType = "postgres"
	StorageTypeInMemory          StorageType = "inMemory"
	PostgresDefaultMigrationsDir             = "internal/db/migrations/"
)

// PostgresParams параметры подключения к PostgreSQL.
type PostgresParams struct {
	DSN           string // Строка подключения
	MigrationsDir string // Директория с миграциями
}

// FactoryConfig конфигурация фабрики подключений.
type FactoryConfig struct {
	StorageType    StorageType     // Тип хранилища
	PostgresParams *PostgresParams // Параметры PostgreSQL
}

// NewConnectionFactory создает новое подключение к хранилищу данных.
//
// Параметры:
//   - ctx: контекст выполнения
//   - config: конфигурация подключения
//
// Возвращает:
//   - any: интерфейс подключения к хранилищу (PostgreSQL pool или MStorage)
//   - error: ошибка создания подключения
func NewConnectionFactory(ctx context.Context, config FactoryConfig) (any, error) {
	switch config.StorageType {
	case StorageTypePostgres:
		if config.PostgresParams == nil {
			return nil, errors.New("postgres config is empty")
		} else if config.PostgresParams.DSN == "" {
			return nil, errors.New("postgres dsn is empty")
		}
		pool, err := NewPostgresConnection(ctx, config.PostgresParams.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres connection: %w", err)
		}
		// Перед инициализацией postgres, нужно убедится что выполнены все миграции
		migrationsDir := config.PostgresParams.MigrationsDir
		if migrationsDir == "" {
			migrationsDir = PostgresDefaultMigrationsDir
		}

		migrateErr := postgresMigrate(migrationsDir, config.PostgresParams.DSN)
		if migrateErr != nil {
			return nil, fmt.Errorf("postgres migration: %w", migrateErr)
		}
		return pool, nil
	case StorageTypeInMemory:
		return NewMemStorage(), nil
	default:
		return nil, fmt.Errorf("unknown storage type: %s", config.StorageType)
	}
}

// postgresMigrate выполняет миграции схемы базы данных PostgreSQL.
//
// Параметры:
//   - dir: путь к директории с файлами миграций
//   - dsn: строка подключения к PostgreSQL
//
// Возвращает:
//   - error: ошибка выполнения миграций
func postgresMigrate(dir string, dsn string) error {
	m, mErr := migrate.New("file://"+dir, dsn)
	if mErr != nil {
		return fmt.Errorf("failed to create migrate instance: %w", mErr)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}
	return nil
}
