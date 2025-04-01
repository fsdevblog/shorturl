package controllers

import (
	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/gin-gonic/gin"
)

func SetupRouter(urlService URLShortener, appConf *config.Config) *gin.Engine {
	r := gin.Default()

	shortURLController := NewShortURLController(urlService, appConf.BaseURL)

	r.GET("/:shortID", shortURLController.Redirect)
	r.POST("/", shortURLController.CreateShortURL)

	return r
}
