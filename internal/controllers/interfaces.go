package controllers

import (
	"context"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/services"
)

//go:generate mockgen -source=interfaces.go -destination=mocksctrl/store.go -package=mocksctrl

type ConnectionChecker interface {
	CheckConnection(ctx context.Context) error
}

type ShortURLStore interface {
	BatchCreate(ctx context.Context, visitorUUID *string, rawURLs []string) (*services.BatchCreateShortURLsResponse, error)
	// Create создает запись models.URL. Возвращает модель, булево значение новая записи или нет и ошибку.
	Create(ctx context.Context, visitorUUID *string, rawURL string) (*models.URL, bool, error)
	GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error)
	GetByURL(ctx context.Context, rawURL string) (*models.URL, error)
	GetAllByVisitorUUID(ctx context.Context, visitorUUID string) ([]models.URL, error)
}
