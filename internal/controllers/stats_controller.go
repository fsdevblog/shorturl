package controllers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// StatsController хендлер статистики.
type StatsController struct {
	svc StatsProvider
}

// NewStatsController создает новый экземпляр StatsController.
//
// service - интерфейс хранилища, который предоставляет методы для получения статистики.
func NewStatsController(service StatsProvider) *StatsController {
	return &StatsController{
		svc: service,
	}
}

// StatsResponse структура для ответа со статистикой.
//
// URLs  - количество сокращенных ссылок.
// Users - количество пользователей.
type StatsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// GetStats обрабатывает запрос на получение статистики.
//
// В случае успешного выполнения возвращает JSON с количеством сокращенных URL и пользователей.
// В случае ошибки возвращает 500 Internal Server Error.
func (s *StatsController) GetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()
	stats, errStats := s.svc.GetStats(ctx)
	if errStats != nil {
		_ = c.Error(errStats)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, StatsResponse{
		URLs:  stats.URLs,
		Users: stats.Users,
	})
}
