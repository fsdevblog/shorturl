package db

import (
	"github.com/fsdevblog/shorturl/internal/app/db/mstorage"
)

type MemoryStorage struct {
	*mstorage.MStorage
}

func NewMemStorage() *MemoryStorage {
	return &MemoryStorage{
		MStorage: mstorage.NewMemStorage(),
	}
}
