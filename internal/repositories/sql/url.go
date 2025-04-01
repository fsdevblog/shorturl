package sql

import (
	"strings"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type URLRepo struct {
	db     *gorm.DB
	logger *logrus.Entry
}

func NewURLRepo(db *gorm.DB, logger *logrus.Logger) *URLRepo {
	return &URLRepo{
		db:     db,
		logger: logger.WithField("module", "repository/sql/url"),
	}
}

func (u *URLRepo) Create(sURL *models.URL) error {
	if err := u.db.Create(sURL).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return repositories.ErrDuplicateKey
		}
		u.logger.WithError(err).Errorf("failed to create record %+v", *sURL)
		return repositories.ErrUnknown
	}
	return nil
}

func (u *URLRepo) GetByShortIdentifier(shortID string) (*models.URL, error) {
	var url models.URL
	if err := u.db.Where("short_identifier = ?", shortID).First(&url).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		u.logger.WithError(err).Errorf("failed to get record by short identifier %s", shortID)
		return nil, errors.Wrapf(err, "failed to get record by short identifier %s", shortID)
	}
	return &url, nil
}

func (u *URLRepo) GetByURL(rawURL string) (*models.URL, error) {
	var url models.URL
	if err := u.db.Where("url = ?", rawURL).First(&url).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		u.logger.WithError(err).Errorf("failed to get record by raw url %s", rawURL)
		return nil, repositories.ErrUnknown
	}
	return &url, nil
}
