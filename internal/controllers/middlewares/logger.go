package middlewares

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware создает middleware для логирования HTTP запросов.
// Должен быть первым в цепочке middleware для корректного логирования всех этапов обработки запроса.
//
// Логирует следующую информацию:
//   - URI запроса
//   - Время обработки запроса (latency)
//   - HTTP статус ответа
//   - HTTP метод
//   - Content-Type заголовок
//   - Content-Encoding заголовок
//   - Accept-Encoding заголовок
//   - Ошибки, возникшие при обработке запроса
//
// Уровни логирования:
//   - ERROR: для статусов 5xx
//   - WARN: для статусов 4xx
//   - INFO: для остальных статусов
//
// Параметры:
//   - logger: экземпляр zap.Logger для логирования
//
// Возвращает:
//   - gin.HandlerFunc: middleware функция
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if logger == nil {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		statusCode := c.Writer.Status()
		l := logger.With(
			zap.String("URI", c.Request.RequestURI),
			zap.String("latency", fmt.Sprintf("%d ms", latency.Milliseconds())),
			zap.Int("status", statusCode),
			zap.String("method", c.Request.Method),
			zap.String("content-type", c.Request.Header.Get("Content-Type")),
			zap.String("content-encoding", c.Request.Header.Get("Content-Encoding")),
			zap.String("accept-encoding", c.Request.Header.Get("Accept-Encoding")),
		)
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if errorMessage != "" {
			l = l.With(zap.String("error", errorMessage))
		}

		switch {
		case statusCode >= http.StatusInternalServerError:
			l.Error("Server error")
		case statusCode >= http.StatusBadRequest:
			l.Warn("Client error")
		default:
			l.Info("Request processed")
		}
	}
}
