package services

import (
	"errors"

	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/repositories/memstore"
	"github.com/fsdevblog/shorturl/internal/repositories/sql"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ServiceType string

const (
	ServiceTypeSQLite   ServiceType = "sqlite"
	ServiceTypeInMemory ServiceType = "inMemory"
)

type Services struct {
	URLService URLShortener
}

func Factory(conn any, sType ServiceType, logger *logrus.Logger) (*Services, error) {
	switch sType {
	case ServiceTypeSQLite:
		var gormDB *gorm.DB
		var ok bool
		if gormDB, ok = conn.(*gorm.DB); !ok {
			return nil, errors.New("invalid connection. expected *gorm.DB")
		}
		return getSQLServices(gormDB, logger), nil
	case ServiceTypeInMemory:
		return getInMemoryServices(logger), nil
	default:
		return nil, errors.New("unknown service type")
	}
}

func getSQLServices(conn *gorm.DB, logger *logrus.Logger) *Services {
	urlRepo := sql.NewURLRepo(conn, logger)
	return &Services{
		URLService: NewURLService(urlRepo),
	}
}

func getInMemoryServices(logger *logrus.Logger) *Services {
	store := db.NewMemStorage()
	urlRepo := memstore.NewURLRepo(store, logger)
	return &Services{
		URLService: NewURLService(urlRepo),
	}
}
