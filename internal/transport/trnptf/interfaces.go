package trnptf

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks
import (
	"context"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/services"
)

// ConnectionChecker определяет интерфейс для проверки соединения с базой данных.
type ConnectionChecker interface {
	// CheckConnection проверяет доступность соединения с базой данных.
	// Возвращает error в случае проблем с соединением.
	CheckConnection(ctx context.Context) error
}

// URLProvider определяет интерфейс сервиса работы со ссылками.
type URLProvider interface {
	// BatchCreate делает пакетную вставку нескольких URL.
	BatchCreate(ctx context.Context, visitorUUID string, rawURLs []string) (*services.BatchCreateShortURLsResponse, error)
	// Create создает запись models.URL. Возвращает короткую ссылку, булево значение новая запись или нет и ошибку.
	Create(ctx context.Context, visitorUUID string, rawURL string) (*models.URL, bool, error)
	// GetByShortIdentifier возвращает оригинальный URL по его короткому идентификатору.
	GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error)
	// GetByURL ищет запись по её URL.
	GetByURL(ctx context.Context, rawURL string) (*models.URL, error)
	// GetAllByVisitorUUID возвращает все URL, созданные определенным посетителем.
	GetAllByVisitorUUID(ctx context.Context, visitorUUID string) ([]models.URL, error)
	// MarkAsDeleted помечает указанные URL как удаленные.
	MarkAsDeleted(ctx context.Context, shortIDs []string, visitorUUID string) error
}

// StatsProvider определяет интерфейс сервиса работы со статистикой.
type StatsProvider interface {
	GetStats(ctx context.Context) (*services.Stats, error)
}
