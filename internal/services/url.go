package services

import (
	"crypto/md5" //nolint:gosec
	"encoding/base64"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"

	"github.com/pkg/errors"
)

type URLRepository interface {
	// Create вычисляет хеш короткой ссылки и создает запись в хранилище.
	Create(rawURL *models.URL) error
	// GetByShortIdentifier находит в хранилище запись по заданному хешу ссылки
	GetByShortIdentifier(shortID string) (*models.URL, error)
	// GetByURL находит запись в хранилище по заданной ссылке
	GetByURL(rawURL string) (*models.URL, error)
}

// URLService Сервис работает с базой данных в контексте таблицы `urls`.
type URLService struct {
	urlRepo URLRepository
}

func NewURLService(urlRepo URLRepository) *URLService {
	return &URLService{urlRepo: urlRepo}
}

func (u *URLService) GetByShortIdentifier(shortID string) (*models.URL, error) {
	sURL, err := u.urlRepo.GetByShortIdentifier(shortID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, errors.Wrapf(ErrRecordNotFound, "id %s not found", shortID)
		}
		return nil, ErrUnknown
	}
	return sURL, nil
}

func (u *URLService) Create(rawURL string) (*models.URL, error) {
	// Мы не можем делать вставку и проверять по ошибке дубликата. Проблема в том, что может быть дубликат как в URL,
	// так и в хеше (коллизия), поэтому сначала мы делаем проверку на существование URL, а только потом делаем
	// вставку

	// В данной реализации не используется система транзакций, т.к. я не знаю как это сделать в моем случае.
	// (не уверен что понимаю как провести транзакции в сервисный слой не высовывая при этом наружу тот самый *gorm.DB,
	// но ещё больше я не уверен в том как реализовать систему транзакция в картах.
	// Но она (система транзакций) тут явно нужна по идее.

	existingURL, existingURLErr := u.urlRepo.GetByURL(rawURL)
	if existingURLErr == nil {
		return existingURL, nil
	}

	var delta uint = 1
	var deltaMax uint = 10

	var sURL models.URL
	for {
		if delta >= deltaMax {
			return nil, errors.Wrap(ErrUnknown, "generateShortID loop limit for url")
		}
		sURL = models.URL{
			URL:             rawURL,
			ShortIdentifier: generateShortID(rawURL, delta, models.ShortIdentifierLength),
		}
		if createErr := u.urlRepo.Create(&sURL); createErr != nil {
			if errors.Is(createErr, repositories.ErrDuplicateKey) {
				delta++
				continue
			}
			return nil, ErrUnknown
		}
		return &sURL, nil
	}
}

// generateShortID генерирует идентификатор для ссылки нужной длины на основе delta.
func generateShortID(rawURL string, delta uint, length int) string {
	// Добавляем счетчик к срезу (для избежания коллизий)
	b := []byte(rawURL)
	b = append(b, byte(delta))

	// Создаем хеш и конвертим в base62
	hash := md5.Sum(b) //nolint:gosec
	base62 := base64.URLEncoding.EncodeToString(hash[:])
	return base62[:length]
}
