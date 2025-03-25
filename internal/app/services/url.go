package services

import (
	"crypto/md5" //nolint:gosec
	"encoding/base64"
	"strings"

	"github.com/fsdevblog/shorturl/internal/app/models"
	"github.com/fsdevblog/shorturl/internal/app/repositories"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// urlService Сервис работающий с базой данных в контексте таблицы `urls`.
type urlService struct {
	urlRepo repositories.IUrl
}

func (u *urlService) GetByShortIdentifier(shortID string) (*models.URL, error) {
	sURL, err := u.urlRepo.GetByShortIdentifier(shortID)
	if err != nil {
		return nil, err // nolint:wrapcheck
	}
	return sURL, nil
}

// Create генерирует short_identifier для урла и сохраняет ссылку в БД.
func (u *urlService) Create(rawURL string) (*models.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	var delta uint = 1
	var sURL models.URL

	/*
		Мы не можем использовать какой-нибудь метод GORM по типу FirstOrCreate так как помимо того
		что ссылка уже может существовать в бд, еще может быть коллизия хеша (комбо 2 в 1),
		поэтому приходится сначала проверять ручками, а затем создавать если приходится
	*/
	txErr := u.urlRepo.ExecuteTransaction(func(tx *gorm.DB) error {
		// если ссылка уже существует - возвращаем её
		existingURL, existingErr := u.urlRepo.WithTx(tx).GetByURL(rawURL)
		if existingErr == nil {
			sURL = *existingURL
			return nil
		}

		// Тут мы всегда ожидаем получить ошибку `gorm.ErrRecordNotFound`. Если любая другая - возвращаем её.
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return errors.Wrap(existingErr, "checking is url exist error")
		}

		// При генерации идентификатора могут возникать ошибки уникальности (коллизии и тд)
		// в таких случаях мы добавляем к хешу счетчик

		var deltaMax uint = 10
		for {
			// Обезопасимся от вечного цикла
			if delta >= deltaMax {
				return errors.New("generateShortIdentifier loop limit")
			}

			sURL = models.URL{
				URL:             rawURL,
				ShortIdentifier: u.generateShortIdentifier(rawURL, delta),
			}
			if err := u.urlRepo.WithTx(tx).Create(&sURL); err != nil {
				// Тут мне самому не ясно, но проверка errors.Is на gorm.ErrDuplicatedKey - не работает
				if strings.Contains(err.Error(), "duplicate") {
					delta++
					continue
				}
				return errors.Wrapf(err, "failed to create url `%s`", rawURL)
			}
			break
		}
		return nil
	})

	if txErr != nil {
		return nil, errors.Wrapf(txErr, "failed to create url `%s`", rawURL)
	}

	return &sURL, nil
}

// generateShortIdentifier генерирует идентификатор для ссылки.
func (u *urlService) generateShortIdentifier(rawURL string, delta uint) string {
	// Добавляем счетчик к срезу (для избежания коллизий)
	b := []byte(rawURL)
	b = append(b, byte(delta))

	// Создаем хеш и конвертим в base62
	hash := md5.Sum(b) //nolint:gosec
	base62 := base64.URLEncoding.EncodeToString(hash[:])
	return base62[:models.ShortIdentifierLength]
}

func NewURLService(urlRepo repositories.IUrl) IURLService {
	return &urlService{urlRepo: urlRepo}
}
