package middlewares

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

// InternalAccess контролирует доступ к ресурсам доступным только из указанной подсети. Если подсеть не задана,
// всегда возвращает HTTP статус ответа 403.
func InternalAccess(trustedSubnet string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if trustedSubnet == "" {
			_ = c.Error(errors.New("internal access middleware: configuration of trusted subnet is empty")).
				SetType(gin.ErrorTypePrivate)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		_, trustedSubnetNet, errParseCIDR := net.ParseCIDR(trustedSubnet)
		if errParseCIDR != nil {
			_ = c.Error(fmt.Errorf("internal access middleware: %w", errParseCIDR)).
				SetType(gin.ErrorTypePrivate)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		clientIP := net.ParseIP(c.ClientIP())
		if clientIP == nil || !trustedSubnetNet.Contains(clientIP) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
