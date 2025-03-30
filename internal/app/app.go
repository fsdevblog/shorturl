package app

import (
	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/controllers"
	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/services"
	"github.com/pkg/errors"
)

type App struct {
	config     *config.Config
	dbServices *services.Services
}

func New(config *config.Config) (*App, error) {
	dbServices, servicesErr := initServices(config)

	if servicesErr != nil {
		return nil, errors.Wrap(servicesErr, "init services error")
	}

	return &App{
		config:     config,
		dbServices: dbServices,
	}, nil
}

// Must вызывает панику если произошла ошибка.
func Must(a *App, err error) *App {
	if err != nil {
		panic(err)
	}
	return a
}

// Run запускает web сервер.
func (a *App) Run() error {
	server := controllers.SetupRouter(a.dbServices.URLService, a.config)
	if serverErr := server.Run(a.config.ServerAddress); serverErr != nil {
		return errors.Wrap(serverErr, "run server error")
	}
	return nil
}

// initServices создает подключение к базе данных и возвращает сервисный слой приложения.
func initServices(appConf *config.Config) (*services.Services, error) {
	dbConn, connErr := db.NewConnection(db.StorageType(appConf.DBType))
	if connErr != nil {
		return nil, connErr //nolint:wrapcheck
	}

	dbServices, dbServErr := services.Factory(dbConn, services.ServiceType(appConf.DBType), appConf.Logger)
	if dbServErr != nil {
		return nil, dbServErr //nolint:wrapcheck
	}
	return dbServices, nil
}
