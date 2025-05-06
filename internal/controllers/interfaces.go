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
	BatchCreate(ctx context.Context, rawURLs []string) (*services.BatchCreateShortURLsResponse, error)
	Create(ctx context.Context, rawURL string) (*models.URL, error)
	GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error)
	GetByURL(ctx context.Context, rawURL string) (*models.URL, error)
}
