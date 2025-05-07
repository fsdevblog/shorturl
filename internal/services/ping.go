package services

import (
	"context"
	"fmt"
)

type Pinger interface {
	Ping(ctx context.Context) error
}
type PingService struct {
	conn Pinger
}

func NewPingService(conn Pinger) *PingService {
	return &PingService{conn: conn}
}

func (s *PingService) CheckConnection(ctx context.Context) error {
	if err := s.conn.Ping(ctx); err != nil {
		return fmt.Errorf("ping error: %w", err)
	}
	return nil
}
