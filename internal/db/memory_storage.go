package db

import (
	"github.com/fsdevblog/shorturl/internal/db/memory"
)

type MemoryStorage struct {
	*memory.MStorage
}

func NewMemStorage() *MemoryStorage {
	return &MemoryStorage{
		MStorage: memory.NewMemStorage(),
	}
}
