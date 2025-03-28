package app

import (
	"github.com/fsdevblog/shorturl/internal/app/config"
	"github.com/fsdevblog/shorturl/internal/app/controllers"
	"github.com/fsdevblog/shorturl/internal/app/services"
	"github.com/gin-gonic/gin"
)

type App struct {
	config     *config.Config
	dbServices *services.Services
}

func NewApp(config *config.Config, dbServices *services.Services) *App {
	return &App{
		config:     config,
		dbServices: dbServices,
	}
}

func (a *App) Run() {
	server := initServer(a.dbServices, a.config)
	if err := server.Run(a.config.ServerAddress); err != nil {
		panic(err)
	}
}

func initServer(services *services.Services, appConf *config.Config) *gin.Engine {
	return controllers.SetupRouter(services.URLService, appConf)
}
