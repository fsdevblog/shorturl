package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PingController контроллер для проверки работоспособности сервиса.
type PingController struct {
	conn ConnectionChecker // Проверяет соединение с базой данных
}

// NewPingController создает новый экземпляр PingController.
//
// Параметры:
//   - conn: интерфейс для проверки соединения
//
// Возвращает:
//   - *PingController: новый экземпляр контроллера
func NewPingController(conn ConnectionChecker) *PingController {
	return &PingController{conn: conn}
}

// Ping обрабатывает GET /ping запрос.
// Проверяет работоспособность сервиса и соединение с базой данных.
//
// В случае успеха возвращает:
//   - HTTP 200 OK с телом "pong"
//
// В случае ошибки возвращает:
//   - HTTP 500 Internal Server Error
//
// Параметры:
//   - ctx: контекст Gin
func (c *PingController) Ping(ctx *gin.Context) {
	pingCtx, cancel := context.WithTimeout(ctx, DefaultRequestTimeout)
	defer cancel()
	if err := c.conn.CheckConnection(pingCtx); err != nil {
		_ = ctx.Error(fmt.Errorf("ping error: %w", err))
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.String(http.StatusOK, "pong")
}
