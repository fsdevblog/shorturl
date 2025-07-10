package controllers

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DefaultRequestTimeout таймаут запроса по умолчанию.
const (
	DefaultRequestTimeout = 3 * time.Second
)

// isJSONRequest Определяет тип запроса (json или нет) по заголовку Content-Type.
func isJSONRequest(ctx *gin.Context) bool {
	ct := ctx.Request.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/json")
}
