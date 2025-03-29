package services

import (
	"strings"

	"github.com/fsdevblog/shorturl/internal/app/apperrs"
	"github.com/pkg/errors"

	"github.com/fsdevblog/shorturl/internal/app/models"
	"github.com/fsdevblog/shorturl/internal/app/repositories"
)

type URLRepository interface {
	// Create вычисляет хеш короткой ссылки и создает запись в хранилище.
	Create(rawURL string) (*models.URL, error)
	// GetByShortIdentifier находит в хранилище запись по заданному хешу ссылки
	GetByShortIdentifier(shortID string) (*models.URL, error)
	// GetByURL находит запись в хранилище по заданной ссылке
	GetByURL(rawURL string) (*models.URL, error)
}

// urlService Сервис работает с базой данных в контексте таблицы `urls`.
type urlService struct {
	urlRepo URLRepository
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

func NewURLService(urlRepo URLRepository) URLShortener {
	return &urlService{urlRepo: urlRepo}
}
