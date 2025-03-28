package sql

import (
	"strings"

	"github.com/fsdevblog/shorturl/internal/app/models"
	"github.com/fsdevblog/shorturl/internal/app/repositories"
	"github.com/fsdevblog/shorturl/internal/app/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type urlRepo struct {
	db     *gorm.DB
	logger *logrus.Entry
}

func NewURLRepo(db *gorm.DB, logger *logrus.Logger) repositories.URLRepository {
	return &urlRepo{
		db:     db,
		logger: logger.WithField("module", "repository/sql/url"),
	}
}

func (u *urlRepo) Create(rawURL string) (*models.URL, error) {
	var sURL models.URL
	var delta uint = 1
	var deltaMax uint = 10

	/*
		Мы не можем использовать какой-нибудь метод GORM по типу FirstOrCreate так как помимо того
		что ссылка уже может существовать в бд, еще может быть коллизия хеша (комбо 2 в 1),
		поэтому приходится сначала проверять ручками, а затем создавать если приходится
	*/

	err := u.db.Transaction(func(tx *gorm.DB) error {
		// если ссылка уже существует - возвращаем её
		existingURL, existingErr := u.withTx(tx).GetByURL(rawURL)
		if existingErr == nil {
			sURL = *existingURL
			return nil
		}

		// Тут мы всегда ожидаем получить ошибку `gorm.ErrRecordNotFound`. Если любая другая - возвращаем её.
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			u.logger.WithError(existingErr).Error("checking is url exist error")
			return repositories.ErrUnknown
		}

		// При генерации идентификатора могут возникать ошибки уникальности (коллизии и тд)
		// в таких случаях мы добавляем к хешу счетчик

		for {
			// Обезопасимся от вечного цикла
			if delta >= deltaMax {
				u.logger.Errorf("generateShortIdentifier loop limit for url %s", rawURL)
				return repositories.ErrUnknown
			}

			sURL = models.URL{
				URL:             rawURL,
				ShortIdentifier: utils.GenerateShortID(rawURL, delta, models.ShortIdentifierLength),
			}
			if err := tx.Create(&sURL).Error; err != nil {
				// Тут мне самому не ясно, но проверка errors.Is на gorm.ErrDuplicatedKey - не работает
				if strings.Contains(err.Error(), "duplicate") {
					delta++
					continue
				}
				u.logger.WithError(err).Errorf("failed to create url `%s`", rawURL)
				return repositories.ErrUnknown
			}
			break
		}
		return nil
	})

	if err != nil {
		return nil, repositories.ErrTransaction
	}

	return &sURL, nil
}

func (u *urlRepo) GetByShortIdentifier(shortID string) (*models.URL, error) {
	var url models.URL
	if err := u.db.Where("short_identifier = ?", shortID).First(&url).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repositories.ErrNotFound
		}
		return nil, errors.Wrapf(err, "failed to get record by short identifier %s", shortID)
	}
	return &url, nil
}

func (u *urlRepo) GetByURL(rawURL string) (*models.URL, error) {
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

// withTx вспомогательный метод для работы с sql транзакциями.
func (u *urlRepo) withTx(tx *gorm.DB) repositories.URLRepository {
	return NewURLRepo(tx, u.logger.Logger)
}
