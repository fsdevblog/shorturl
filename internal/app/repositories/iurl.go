package repositories

import (
	"github.com/fsdevblog/shorturl/internal/app/models"
	"gorm.io/gorm"
)

type IUrl interface {
	BaseRepositoryInterface
	Create(url *models.URL) error
	GetByShortIdentifier(shortID string) (*models.URL, error)
	GetByURL(url string) (*models.URL, error)
	WithTx(tx *gorm.DB) IUrl
}
