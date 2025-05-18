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

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data) //nolint:wrapcheck
}

func GzipMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		readGzip(ctx)
		writeGzip(ctx)
		ctx.Next()
	}
}

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

// readGzip Определяет сжаты ли данные gzip. Имеет смысл это делать только для POST|PUT|PATCH запросов
// затем распаковывает gzip и подменяет тело запроса на распакованное.
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
