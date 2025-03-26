package controllers

import (
	"github.com/fsdevblog/shorturl/internal/app/services"
	"github.com/gin-gonic/gin"
)

func SetupRouter(urlService services.IURLService) *gin.Engine {
	r := gin.Default()
	shortURLController := NewShortURLController(urlService)

	r.GET("/:shortID", shortURLController.Redirect)
	r.POST("/", shortURLController.CreateShortURL)

	return r
}
