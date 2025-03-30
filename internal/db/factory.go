package db

import (
	"fmt"
)

type StorageType string

const (
	StorageTypeSQLite   StorageType = "sqlite"
	StorageTypeInMemory StorageType = "inMemory"
)

const SQLiteDBPath = "./shortener.sqlite"

func NewConnection(storageType StorageType) (any, error) {
	switch storageType {
	case StorageTypeSQLite:
		return NewSQLite(SQLiteDBPath)
	case StorageTypeInMemory:
		return NewMemStorage(), nil
	default:
		return nil, fmt.Errorf("unknown storage type: %s", storageType)
	}
}
