package controllers

import (
	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/controllers/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func SetupRouter(urlService URLShortener, appConf *config.Config, l *logrus.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middlewares.LoggerMiddleware(l))
	r.Use(middlewares.GzipMiddleware())

	shortURLController := NewShortURLController(urlService, appConf.BaseURL)

	r.GET("/:shortID", shortURLController.Redirect)
	r.POST("/", shortURLController.CreateShortURL)

	api := r.Group("/api")
	api.POST("/shorten", shortURLController.CreateShortURL)
	api.GET("/:shortID", shortURLController.Redirect)
	return r
}
