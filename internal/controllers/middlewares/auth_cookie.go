package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fsdevblog/shorturl/internal/tokens"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	VisitorUUIDKey           = "visitorUUID"
	VisitorCookieName        = "visitor"
	VisitorJWTExpireDuration = 24 * time.Hour
)

func VisitorCookieMiddleware(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		visitorAuthCookie, err := c.Request.Cookie(VisitorCookieName)

		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			// куки не работают. Нам тут делать нечего, отправляем ошибку выше и едем дальше.
			_ = c.Error(fmt.Errorf("visitor cookie middleware: %w", err))
			c.Next()
			return
		}

		var visitorUUID string
		needGenerateJWT := true

		if visitorAuthCookie != nil {
			// Проверяем токен
			token, validateErr := tokens.ValidateVisitorJWT(visitorAuthCookie.Value, jwtSecret)
			if validateErr != nil {
				// отправляем ошибку и будем выставлять новый токен.
				_ = c.Error(fmt.Errorf("visitor cookie middleware: %w", validateErr))
			} else if token.Valid {
				needGenerateJWT = false

				// Безопасная операция, т.к. проверка типа происходит в tokens.ValidateVisitorJWT.
				visitorUUID = token.Claims.(*tokens.VisitorClaims).UUID //nolint:errcheck
			}
		}

		if needGenerateJWT {
			var uErr error
			visitorUUID, uErr = generateUUID()
			if uErr != nil {
				_ = c.Error(fmt.Errorf("visitor cookie middleware: %w", uErr))
				c.Next()
				return
			}
			tokenString, tokenErr := tokens.GenerateVisitorJWT(visitorUUID, VisitorJWTExpireDuration, jwtSecret)
			if tokenErr != nil {
				_ = c.Error(fmt.Errorf("visitor cookie middleware: %w", tokenErr))
				c.Next()
				return
			}
			c.SetCookie(
				VisitorCookieName,
				tokenString,
				int(VisitorJWTExpireDuration.Seconds()),
				"/",
				"",
				false,
				true,
			)
		}

		// Устанавливаем UUID посетителя в контекст gin.
		c.Set(VisitorUUIDKey, visitorUUID)
		c.Next()
	}
}

func generateUUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate uuid error: %w", err)
	}
	return u.String(), nil
}
