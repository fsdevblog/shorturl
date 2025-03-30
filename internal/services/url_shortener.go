package services

import (
	"github.com/fsdevblog/shorturl/internal/models"
)

type URLShortener interface {
	Create(rawURL string) (*models.URL, error)
	GetByShortIdentifier(shortID string) (*models.URL, error)
}
