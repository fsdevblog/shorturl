package services

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/repositories/memstore"
	"github.com/fsdevblog/shorturl/internal/repositories/sql"
)

// ServiceType определяет тип реализации.
type ServiceType string

const (
	ServiceTypePostgres ServiceType = "postgres" // PostgreSQL реализация
	ServiceTypeInMemory ServiceType = "inMemory" // In-memory реализация
)

// Services объединяет все сервисы приложения.
type Services struct {
	URLService  *URLService  // Сервис для работы с URL
	PingService *PingService // Сервис для проверки соединения
}

// Factory создает набор сервисов в зависимости от указанного типа.
//
// Параметры:
//   - conn: соединение с хранилищем данных
//   - sType: тип сервисов (ServiceTypePostgres или ServiceTypeInMemory)
//
// Возвращает:
//   - *Services: инициализированные сервисы
//   - error: ошибка создания сервисов
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

// getSQLServices создает сервисы для работы с PostgreSQL.
//
// Параметры:
//   - conn: пул подключений к PostgreSQL
//
// Возвращает:
//   - *Services: сервисы с PostgreSQL реализацией
func getSQLServices(conn *pgxpool.Pool) *Services {
	urlRepo := sql.NewURLRepo(conn)
	return &Services{
		URLService:  NewURLService(urlRepo),
		PingService: NewPingService(conn),
	}
}

// getInMemoryServices создает сервисы для работы с in-memory хранилищем.
//
// Возвращает:
//   - *Services: сервисы с in-memory реализацией
func getInMemoryServices() *Services {
	store := db.NewMemStorage()
	urlRepo := memstore.NewURLRepo(store)
	return &Services{
		URLService:  NewURLService(urlRepo),
		PingService: NewPingService(store),
	}
}
