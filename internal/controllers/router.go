package controllers

import (
	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/gin-gonic/gin"
)

func SetupRouter(urlService URLShortener, appConf *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware(appConf.Logger))

	shortURLController := NewShortURLController(urlService, appConf.BaseURL)

	r.GET("/:shortID", shortURLController.Redirect)
	r.POST("/", shortURLController.CreateShortURL)

	api := r.Group("/api")
	api.POST("/shorten", shortURLController.CreateShortURL)
	api.GET("/:shortID", shortURLController.Redirect)
	return r
}
