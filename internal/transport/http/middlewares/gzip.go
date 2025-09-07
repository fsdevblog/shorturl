package middlewares

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	headerContentEncoding = "Content-Encoding"
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentLength   = "Content-Length"
	headerVary            = "Vary"
)

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
	pool := newGzipHandler()
	return func(ctx *gin.Context) {
		pool.handleRead(ctx)
		pool.handleWrite(ctx)
	}
}

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
	g.Header().Del("Content-Length")
	return g.writer.Write(data) //nolint:wrapcheck
}

type poolHandler struct {
	writePool sync.Pool
	readPool  sync.Pool
}

func newGzipHandler() *poolHandler {
	return &poolHandler{
		writePool: sync.Pool{
			New: func() any {
				// используем gzip.DefaultCompression
				return gzip.NewWriter(nil)
			},
		},
		readPool: sync.Pool{
			New: func() any {
				return new(gzip.Reader)
			},
		}}
}

func (g *poolHandler) handleWrite(c *gin.Context) {
	if !strings.Contains(c.Request.Header.Get(headerAcceptEncoding), "gzip") {
		c.Next()
		return
	}

	gzWriter := g.writePool.Get().(*gzip.Writer) //nolint:errcheck
	gzWriter.Reset(c.Writer)
	c.Header(headerContentEncoding, "gzip")
	c.Writer.Header().Add(headerVary, headerAcceptEncoding)

	c.Writer = &gzipWriter{
		ResponseWriter: c.Writer,
		writer:         gzWriter,
	}
	defer func() {
		if c.Writer.Size() < 0 {
			// если размер -1, значит в тело ответа ничего не записали
			// поэтому нет смысла записывать gzip footer
			gzWriter.Reset(io.Discard)
		}

		if closeErr := gzWriter.Close(); closeErr != nil {
			_ = c.Error(fmt.Errorf("close gzip writer: %s", closeErr.Error()))
		}

		if c.Writer.Size() >= 0 {
			// Устанавливаем заголовок с размером ответа
			c.Header(headerContentLength, strconv.Itoa(c.Writer.Size()))
		}

		// возвращаем в пул
		g.writePool.Put(gzWriter)
	}()
	c.Next()
}

func (g *poolHandler) handleRead(c *gin.Context) {
	if !slices.Contains([]string{http.MethodPost, http.MethodPut, http.MethodPatch}, c.Request.Method) {
		return
	}

	if !strings.Contains(c.Request.Header.Get(headerContentEncoding), "gzip") {
		return
	}

	gzReader := g.readPool.Get().(*gzip.Reader) //nolint:errcheck
	if errReset := gzReader.Reset(c.Request.Body); errReset != nil {
		// возвращаем в пул
		g.readPool.Put(gzReader)

		_ = c.Error(fmt.Errorf("reset gzip reader: %s", errReset.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	defer func() {
		if errClose := gzReader.Close(); errClose != nil {
			_ = c.Error(fmt.Errorf("close gzip reader: %s", errClose.Error()))
		}
		g.readPool.Put(gzReader)
	}()
	bodyBytes, err := io.ReadAll(gzReader)
	if err != nil {
		_ = c.Error(fmt.Errorf("read gzip: %s", err.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
}
