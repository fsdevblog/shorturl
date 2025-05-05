package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate mockgen -source=ping_controller.go -destination=mocksctrl/pinger.go -package=mocksctrl
type ConnectionChecker interface {
	CheckConnection(ctx context.Context) error
}

type PingController struct {
	conn ConnectionChecker
}

func NewPingController(conn ConnectionChecker) *PingController {
	return &PingController{conn: conn}
}

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
