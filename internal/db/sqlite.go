package db

import (
	"fmt"

	"github.com/fsdevblog/shorturl/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewSQLite(dbPath string) (*gorm.DB, error) {
	conn, connErr := connectSQLite(dbPath)
	if connErr != nil {
		return nil, fmt.Errorf("init database error: %w", connErr)
	}
	if migrateErr := migrateSQLite(conn); migrateErr != nil {
		return nil, fmt.Errorf("migrate database error: %w", migrateErr)
	}
	return conn, nil
}

func connectSQLite(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("connect database with path %s error: %w", dbPath, err)
	}
	return db, nil
}

func migrateSQLite(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.URL{}); err != nil {
		return fmt.Errorf("migrating sql: %w", err)
	}
	return nil
}
