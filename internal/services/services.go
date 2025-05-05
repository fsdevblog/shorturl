package services

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/repositories/memstore"
	"github.com/fsdevblog/shorturl/internal/repositories/sql"
)

type ServiceType string

const (
	ServiceTypePostgres ServiceType = "postgres"
	ServiceTypeInMemory ServiceType = "inMemory"
)

type Services struct {
	URLService  *URLService
	PingService *PingService
}

func Factory(conn any, sType ServiceType) (*Services, error) {
	switch sType {
	case ServiceTypePostgres:
		pool, ok := conn.(*pgxpool.Pool)
		if !ok {
			return nil, errors.New("invalid connection type. expected *pgxpool.Pool")
		}
		return getSQLServices(pool), nil
	case ServiceTypeInMemory:
		return getInMemoryServices(), nil
	default:
		return nil, fmt.Errorf("unknown service type: %s", sType)
	}
}

func getSQLServices(conn *pgxpool.Pool) *Services {
	urlRepo := sql.NewURLRepo(conn)
	return &Services{
		URLService:  NewURLService(urlRepo),
		PingService: NewPingService(conn),
	}
}

func getInMemoryServices() *Services {
	store := db.NewMemStorage()
	urlRepo := memstore.NewURLRepo(store)
	return &Services{
		URLService:  NewURLService(urlRepo),
		PingService: NewPingService(store),
	}
}
