package middlewares

import (
	"fmt"
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

// VisitorCookieMiddleware создает middleware для аутентификации посетителей через cookie.
// Проверяет наличие и валидность JWT токена в cookie. Если токен отсутствует или невалиден,
// генерирует новый UUID посетителя и создает новый JWT токен.
//
// Алгоритм работы:
//  1. Проверяет наличие cookie с JWT токеном
//  2. Если токен есть - проверяет его валидность
//  3. Если токен отсутствует или невалиден - генерирует новый UUID и JWT токен
//  4. Сохраняет UUID посетителя в контексте запроса
//  5. При необходимости устанавливает новую cookie с JWT токеном
//
// Параметры:
//   - jwtSecret: секретный ключ для подписи JWT токенов
//
// Возвращает:
//   - gin.HandlerFunc: middleware функция
//
// Устанавливает в контексте:
//   - VisitorUUIDKey: UUID посетителя (string)
func VisitorCookieMiddleware(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		visitorAuthCookie, _ := c.Request.Cookie(VisitorCookieName)

		var visitorUUID string
		needGenerateJWT := true

		if visitorAuthCookie != nil {
			// Проверяем токен
			token, validateErr := tokens.ValidateVisitorJWT(visitorAuthCookie.Value, jwtSecret)
			if validateErr != nil {
				// отправляем ошибку и будем выставлять новый токен.
				_ = c.Error(fmt.Errorf("visitor cookie middleware: %s", validateErr.Error()))
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
				_ = c.Error(fmt.Errorf("visitor cookie middleware: %s", uErr.Error()))
				c.Next()
				return
			}
			tokenString, tokenErr := tokens.GenerateVisitorJWT(visitorUUID, VisitorJWTExpireDuration, jwtSecret)
			if tokenErr != nil {
				_ = c.Error(fmt.Errorf("visitor cookie middleware: %s", tokenErr.Error()))
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

// generateUUID генерирует случайный UUID v4.
//
// Возвращает:
//   - string: сгенерированный UUID в строковом представлении
//   - error: ошибка генерации UUID
func generateUUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate uuid error: %s", err.Error())
	}
	return u.String(), nil
}
