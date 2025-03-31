package memstore

import (
	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/db/memory"
	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type URLRepo struct {
	s      *db.MemoryStorage
	logger *logrus.Entry
}

func NewURLRepo(store *db.MemoryStorage, logger *logrus.Logger) *URLRepo {
	return &URLRepo{
		s:      store,
		logger: logger.WithField("module", "repository/memstore/url"),
	}
}

func (u *URLRepo) Create(sURL *models.URL) error {
	if err := memory.Set[models.URL](sURL.ShortIdentifier, sURL, u.s.MStorage); err != nil {
		if errors.Is(err, memory.ErrDuplicateKey) {
			return repositories.ErrDuplicateKey
		}
		return errors.Wrapf(repositories.ErrUnknown, "failed to create record %+v", *sURL)
	}
	return nil
}

func (u *URLRepo) GetByShortIdentifier(shortID string) (*models.URL, error) {
	url, err := memory.Get[models.URL](shortID, u.s.MStorage)
	if err != nil {
		if errors.Is(err, memory.ErrNotFound) {
			return nil, repositories.ErrNotFound
		}
		u.logger.WithError(err).Errorf("failed to get record by short identifier %s", shortID)
		return nil, repositories.ErrUnknown
	}
	return url, nil
}

func (u *URLRepo) GetByURL(rawURL string) (*models.URL, error) {
	data := memory.GetAll[models.URL](u.s.MStorage)

	for _, val := range data {
		if val.URL == rawURL {
			return &val, nil
		}
	}
	return nil, repositories.ErrNotFound
}
