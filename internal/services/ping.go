package services

import (
	"context"
	"fmt"
)

// Pinger определяет интерфейс для проверки соединения.
type Pinger interface {
	// Ping проверяет доступность соединения.
	//
	// Параметры:
	//   - ctx: контекст выполнения
	//
	// Возвращает:
	//   - error: ошибка проверки соединения
	Ping(ctx context.Context) error
}

// PingService предоставляет функционал для проверки соединения.
type PingService struct {
	conn Pinger
}

// NewPingService создает новый экземпляр сервиса проверки соединения.
//
// Параметры:
//   - conn: реализация интерфейса Pinger
//
// Возвращает:
//   - *PingService: инициализированный сервис
func NewPingService(conn Pinger) *PingService {
	return &PingService{conn: conn}
}

// CheckConnection выполняет проверку соединения.
//
// Параметры:
//   - ctx: контекст выполнения
//
// Возвращает:
//   - error: ошибка проверки соединения
func (s *PingService) CheckConnection(ctx context.Context) error {
	if err := s.conn.Ping(ctx); err != nil {
		return fmt.Errorf("ping error: %w", err)
	}
	return nil
}
