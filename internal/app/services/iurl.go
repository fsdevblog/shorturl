package services

import "github.com/fsdevblog/shorturl/internal/app/models"

type IURLService interface {
	Create(rawURL string) (*models.URL, error)
	GetByShortIdentifier(shortId string) (*models.URL, error)
}
