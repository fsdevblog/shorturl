package db

import (
	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewSQLite(dbPath string) (*gorm.DB, error) {
	conn, connErr := connectSQLite(dbPath)
	if connErr != nil {
		return nil, errors.Wrap(connErr, "init database error")
	}
	if migrateErr := migrateSQLite(conn); migrateErr != nil {
		return nil, errors.Wrap(migrateErr, "migrate database error")
	}
	return conn, nil
}

func connectSQLite(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to sql db `%s`", dbPath)
	}
	return db, nil
}

func migrateSQLite(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.URL{}); err != nil {
		return errors.Wrap(err, "migrating sql failed")
	}
	return nil
}
