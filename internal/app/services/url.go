package services

import (
	"strings"

	"github.com/fsdevblog/shorturl/internal/app/apperrs"
	"github.com/pkg/errors"

	"github.com/fsdevblog/shorturl/internal/app/models"
	"github.com/fsdevblog/shorturl/internal/app/repositories"
)

// urlService Сервис работает с базой данных в контексте таблицы `urls`.
type urlService struct {
	urlRepo repositories.URLRepository
}

func (u *urlService) GetByShortIdentifier(shortID string) (*models.URL, error) {
	sURL, err := u.urlRepo.GetByShortIdentifier(shortID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, apperrs.ErrRecordNotFound
		}
		return nil, apperrs.ErrInternal
	}
	return sURL, nil
}

func (u *urlService) Create(rawURL string) (*models.URL, error) {
	record, err := u.urlRepo.Create(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, apperrs.ErrInternal
	}

	return record, nil
}

func NewURLService(urlRepo repositories.URLRepository) URLShortener {
	return &urlService{urlRepo: urlRepo}
}
