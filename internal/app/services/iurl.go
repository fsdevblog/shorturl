package services

import "github.com/fsdevblog/shorturl/internal/app/models"

type IURLService interface {
	Create(rawURL string) (*models.URL, error)
	GetByShortIdentifier(shortID string) (*models.URL, error)
}
