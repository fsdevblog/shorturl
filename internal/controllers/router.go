package controllers

import (
	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/controllers/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type RouterParams struct {
	URLService  ShortURLStore
	PingService ConnectionChecker
	AppConf     config.Config
	Logger      *logrus.Logger
}

func SetupRouter(params RouterParams) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	if params.Logger != nil {
		r.Use(middlewares.LoggerMiddleware(params.Logger))
	}

	r.Use(middlewares.GzipMiddleware())

	shortURLController := NewShortURLController(params.URLService, params.AppConf.BaseURL)
	pingController := NewPingController(params.PingService)

	r.GET("/:shortID", shortURLController.Redirect)
	r.POST("/", shortURLController.CreateShortURL)
	r.GET("/ping", pingController.Ping)

	api := r.Group("/api")
	api.POST("/shorten", shortURLController.CreateShortURL)
	api.POST("/shorten/batch", shortURLController.BatchCreate)
	api.GET("/:shortID", shortURLController.Redirect)
	return r
}
