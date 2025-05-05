package memstore

import (
	"context"
	"fmt"

	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/db/memory"
	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"
	"github.com/pkg/errors"
)

type URLRepo struct {
	s *db.MemoryStorage
}

func NewURLRepo(store *db.MemoryStorage) *URLRepo {
	return &URLRepo{
		s: store,
	}
}

func (u *URLRepo) Create(ctx context.Context, sURL *models.URL) error {
	if err := memory.Set[models.URL](ctx, sURL.ShortIdentifier, sURL, u.s.MStorage); err != nil {
		if errors.Is(err, memory.ErrDuplicateKey) {
			return repositories.ErrDuplicateKey
		}
		return fmt.Errorf(
			"%w: failed to create record: %s",
			repositories.ErrUnknown, err.Error())
	}
	return nil
}

func (u *URLRepo) GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error) {
	url, err := memory.Get[models.URL](ctx, shortID, u.s.MStorage)
	if err != nil {
		if errors.Is(err, memory.ErrNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, fmt.Errorf(
			"%w: failed to get record by short identifier %s",
			repositories.ErrUnknown, shortID,
		)
	}
	return url, nil
}

func (u *URLRepo) GetByURL(ctx context.Context, rawURL string) (*models.URL, error) {
	data, err := memory.GetAll[models.URL](ctx, u.s.MStorage)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed to get record by url %s records: %s",
			repositories.ErrUnknown, rawURL, err.Error(),
		)
	}

	for _, val := range data {
		if val.URL == rawURL {
			return &val, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (u *URLRepo) GetAll(ctx context.Context) ([]models.URL, error) {
	urls, err := memory.GetAll[models.URL](ctx, u.s.MStorage)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: failed to get all records: %s",
			repositories.ErrUnknown, err.Error(),
		)
	}
	return urls, nil
}
