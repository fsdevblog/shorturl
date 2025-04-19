package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func loggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		statusCode := c.Writer.Status()
		l := logger.WithFields(logrus.Fields{
			"URI":          c.Request.RequestURI,
			"latency":      fmt.Sprintf("%d ms", latency.Milliseconds()),
			"status":       statusCode,
			"method":       c.Request.Method,
			"content-type": c.Request.Header.Get("Content-Type"),
		})
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if errorMessage != "" {
			l = l.WithField("error", errorMessage)
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
