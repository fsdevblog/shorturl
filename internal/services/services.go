package services

import (
	"errors"
	"fmt"

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
		gormDB, ok := conn.(*gorm.DB)
		if !ok {
			return nil, errors.New("invalid connection type. expected *gorm.DB")
		}
		return getSQLServices(gormDB, logger), nil
	case ServiceTypeInMemory:
		return getInMemoryServices(logger), nil
	default:
		return nil, fmt.Errorf("unknown service type: %s", sType)
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
