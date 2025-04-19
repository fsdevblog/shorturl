package controllers

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// isJSONRequest Определяет тип запроса (json или нет) по заголовку Content-Type.
func isJSONRequest(ctx *gin.Context) bool {
	ct := ctx.Request.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/json")
}
