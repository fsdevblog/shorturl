package middlewares

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// gzipWriter обертка над gin.ResponseWriter для сжатия ответов в формате gzip.
type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

// Write реализует интерфейс io.Writer.
// Записывает сжатые данные в формате gzip.
//
// Параметры:
//   - data: данные для записи
//
// Возвращает:
//   - int: количество записанных байт
//   - error: ошибка записи
func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data) //nolint:wrapcheck
}

// GzipMiddleware создает middleware для автоматического сжатия ответов
// и распаковки запросов в формате gzip.
//
// Для ответов:
//   - Проверяет поддержку gzip в заголовке Accept-Encoding
//   - При поддержке сжимает ответ и устанавливает соответствующие заголовки
//   - Content-Encoding: gzip
//   - Vary: Accept-Encoding
//
// Для запросов:
//   - Обрабатывает только POST, PUT, PATCH запросы
//   - Проверяет наличие заголовка Content-Encoding: gzip
//   - При наличии распаковывает тело запроса
//
// Возвращает:
//   - gin.HandlerFunc: middleware функция
func GzipMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		readGzip(ctx)
		writeGzip(ctx)
		ctx.Next()
	}
}

// writeGzip настраивает сжатие ответа в формате gzip.
//
// Параметры:
//   - ctx: контекст Gin
func writeGzip(ctx *gin.Context) {
	if !strings.Contains(ctx.Request.Header.Get("Accept-Encoding"), "gzip") {
		ctx.Next()
		return
	}

	ctx.Header("Content-Encoding", "gzip")
	ctx.Header("Vary", "Accept-Encoding")

	gzw := gzip.NewWriter(ctx.Writer)
	defer func() {
		if closeErr := gzw.Close(); closeErr != nil {
			_ = ctx.Error(fmt.Errorf("close gzip writer: %w", closeErr))
		}
	}()

	gzWriter := &gzipWriter{
		ResponseWriter: ctx.Writer,
		writer:         gzw,
	}

	ctx.Writer = gzWriter
	ctx.Next()
}

// readGzip обрабатывает сжатые запросы в формате gzip.
// Распаковывает тело запроса если оно сжато.
//
// Параметры:
//   - ctx: контекст Gin
func readGzip(ctx *gin.Context) {
	if slices.Contains([]string{http.MethodPost, http.MethodPut, http.MethodPatch}, ctx.Request.Method) {
		ce := ctx.Request.Header.Get("Content-Encoding")
		if !strings.Contains(ce, "gzip") {
			return
		}

		gzReader, gzErr := gzip.NewReader(ctx.Request.Body)
		if gzErr != nil {
			_ = ctx.Error(fmt.Errorf("read gzip: %w", gzErr))
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		defer func() {
			if closeErr := gzReader.Close(); closeErr != nil {
				_ = ctx.Error(fmt.Errorf("close gzip reader: %w", closeErr))
			}
		}()
		bodyBytes, err := io.ReadAll(gzReader)
		if err != nil {
			_ = ctx.Error(fmt.Errorf("read gzip: %w", err))
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}

		ctx.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
}
