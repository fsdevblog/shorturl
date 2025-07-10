package services

import (
	"context"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/mock.go -package=mocks

// URLRepository описывает репозиторий для URL.
type URLRepository interface {
	BatchCreate(
		ctx context.Context,
		mURLs []repositories.BatchCreateArg,
	) (*repositories.BatchCreateShortURLsResult, error)

	// Create вычисляет хеш короткой ссылки и создает запись в хранилище.
	// Возвращает два значения: bool отвечает за уникальность созданной записи, 2 ошибку.
	Create(ctx context.Context, mURL *models.URL) (*models.URL, bool, error)
	// GetByShortIdentifier находит в хранилище запись по заданному хешу ссылки
	GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error)
	// GetByURL находит запись в хранилище по заданной ссылке
	GetByURL(ctx context.Context, rawURL string) (*models.URL, error)
	// GetAll возвращает все записи в бд. Сразу пачкой.
	GetAll(ctx context.Context) ([]models.URL, error)
	// GetAllByVisitorUUID возвращает записи связанные с visitorUUID.
	GetAllByVisitorUUID(ctx context.Context, visitorUUID string) ([]models.URL, error)
	// DeleteByShortIDsVisitorUUID помечает записи как удаленные.
	DeleteByShortIDsVisitorUUID(ctx context.Context, visitorUUID string, shortIDs []string) error
}
