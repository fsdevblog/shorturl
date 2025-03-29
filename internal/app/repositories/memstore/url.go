package memstore

import (
	"github.com/fsdevblog/shorturl/internal/app/db"
	"github.com/fsdevblog/shorturl/internal/app/db/mstorage"
	"github.com/fsdevblog/shorturl/internal/app/models"
	"github.com/fsdevblog/shorturl/internal/app/repositories"
	"github.com/fsdevblog/shorturl/internal/app/utils"
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

func (u *URLRepo) Create(rawURL string) (*models.URL, error) {
	return u.recursiveCreate(rawURL, 1)
}

func (u *URLRepo) GetByShortIdentifier(shortID string) (*models.URL, error) {
	url, err := mstorage.Get[models.URL](shortID, u.s.MStorage)
	if err != nil {
		if errors.Is(err, mstorage.ErrNotFound) {
			return nil, repositories.ErrNotFound
		}
		u.logger.WithError(err).Errorf("failed to get record by short identifier %s", shortID)
		return nil, repositories.ErrUnknown
	}
	return url, nil
}

func (u *URLRepo) GetByURL(rawURL string) (*models.URL, error) {
	data := mstorage.GetAll[models.URL](u.s.MStorage)

	for _, val := range data {
		if val.URL == rawURL {
			return &val, nil
		}
	}
	return nil, repositories.ErrNotFound
}

// recursiveCreate вспомогательная рекурсивная функция, возвращает модель с хешем ссылки в обход коллизии.
// Параметр `delta` служит для создания соли хеша и инкрементится с каждой рекурсией.
func (u *URLRepo) recursiveCreate(rawURL string, delta uint) (*models.URL, error) {
	shortID := utils.GenerateShortID(rawURL, delta, models.ShortIdentifierLength)
	existingURL, getErr := mstorage.Get[models.URL](shortID, u.s.MStorage)
	var maxDelta uint = 10
	// если ошибка отличная от ErrNotFound, возвращаем её
	if getErr != nil && !errors.Is(getErr, mstorage.ErrNotFound) {
		return nil, getErr
	}

	if existingURL != nil {
		// если запись уже имеется и нет коллизии, возвращаем
		if existingURL.URL == rawURL {
			return existingURL, nil
		}

		// если запись имеется, но обнаружена коллизия - инкрементим дельту и вызываем рекурсию
		if existingURL.URL != rawURL {
			// обезопасимся от вечной рекурсии
			if delta >= maxDelta {
				u.logger.Errorf("generateShortIdentifier loop limit for url %s", rawURL)
				return nil, repositories.ErrUnknown
			}
			delta++
			return u.recursiveCreate(rawURL, delta)
		}
	}

	url := models.URL{
		ID:              uint(u.s.Len() + 1), //nolint:gosec
		URL:             rawURL,
		ShortIdentifier: shortID,
	}
	if createErr := mstorage.Set[models.URL](shortID, url, u.s.MStorage); createErr != nil {
		return nil, createErr
	}
	return &url, nil
}
