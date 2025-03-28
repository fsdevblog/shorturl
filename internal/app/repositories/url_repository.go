package repositories

import (
	"github.com/fsdevblog/shorturl/internal/app/models"
)

type URLRepository interface {
	// Create вычисляет хеш короткой ссылки и создает запись в хранилище.
	Create(rawURL string) (*models.URL, error)
	// GetByShortIdentifier находит в хранилище запись по заданному хешу ссылки
	GetByShortIdentifier(shortID string) (*models.URL, error)
	// GetByURL находит запись в хранилище по заданной ссылке
	GetByURL(rawURL string) (*models.URL, error)
}
