package sql

import (
	"fmt"

	"github.com/fsdevblog/shorturl/internal/models"
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
		u.logger.WithError(err).Errorf("failed to create record %+v", *sURL)
		return ConvertErrorType(err)
	}
	return nil
}

func (u *URLRepo) GetByShortIdentifier(shortID string) (*models.URL, error) {
	var url models.URL
	if err := u.db.Where("short_identifier = ?", shortID).First(&url).Error; err != nil {
		u.logger.WithError(err).Errorf("failed to get record by short identifier %s", shortID)
		return nil, ConvertErrorType(fmt.Errorf("failed to get record by short identifier %s: %w", shortID, err))
	}
	return &url, nil
}

func (u *URLRepo) GetByURL(rawURL string) (*models.URL, error) {
	var url models.URL
	if err := u.db.Where("url = ?", rawURL).First(&url).Error; err != nil {
		u.logger.WithError(err).Errorf("failed to get record by raw url %s", rawURL)
		return nil, ConvertErrorType(fmt.Errorf("failed to get record by raw url %s: %w", rawURL, err))
	}
	return &url, nil
}

func (u *URLRepo) GetAll() ([]models.URL, error) {
	var urls []models.URL
	if err := u.db.Find(&urls).Error; err != nil {
		u.logger.WithError(err).Errorf("failed to get all records")
		return nil, ConvertErrorType(err)
	}
	return urls, nil
}
