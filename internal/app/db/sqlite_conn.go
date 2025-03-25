package db

import (
	"github.com/fsdevblog/shorturl/internal/app/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func ConnectSQLite(dbPath string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

func MigrateSQLite(db *gorm.DB) {
	if err := db.AutoMigrate(&models.URL{}); err != nil {
		panic(err)
	}
}
