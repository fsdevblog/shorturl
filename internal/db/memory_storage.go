package db

import (
	"github.com/fsdevblog/shorturl/internal/db/memory"
)

// MemoryStorage представляет собой обертку над внутренним in-memory хранилищем.
type MemoryStorage struct {
	*memory.MStorage
}

// NewMemStorage создает новый экземпляр in-memory хранилища.
//
// Возвращает:
//   - *MemoryStorage: инициализированное хранилище в памяти
func NewMemStorage() *MemoryStorage {
	return &MemoryStorage{
		MStorage: memory.NewMemStorage(),
	}
}
