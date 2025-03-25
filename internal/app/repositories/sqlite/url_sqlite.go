package sqlite

import (
	"github.com/fsdevblog/shorturl/internal/app/models"
	"github.com/fsdevblog/shorturl/internal/app/repositories"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type urlRepo struct {
	*BaseRepository
}

func NewURLRepo(db *gorm.DB) repositories.IUrl {
	return &urlRepo{&BaseRepository{db: db}}
}

func (u *urlRepo) WithTx(tx *gorm.DB) repositories.IUrl {
	return NewURLRepo(tx)
}

func (u *urlRepo) Create(url *models.URL) error {
	if err := u.GetDB().Create(url).Error; err != nil {
		return errors.Wrapf(err, "failed to create url %+v", url)
	}
	return nil
}

func (u *urlRepo) GetByShortIdentifier(shortID string) (*models.URL, error) {
	var url models.URL
	if err := u.GetDB().Where("short_identifier = ?", shortID).First(&url).Error; err != nil {
		return nil, errors.Wrapf(err, "failed to get url by short identifier %s", shortID)
	}
	return &url, nil
}

func (u *urlRepo) GetByURL(rawURL string) (*models.URL, error) {
	var url models.URL
	if err := u.GetDB().Where("url = ?", rawURL).First(&url).Error; err != nil {
		return nil, errors.Wrapf(err, "failed to get url by raw url %s", rawURL)
	}
	return &url, nil
}
