package app

import (
	"os"
	"os/signal"
	"syscall"

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

func (a *App) restoreBackup() error {
	if err := a.dbServices.URLService.RestoreBackup(a.config.FileStoragePath); err != nil {
		return errors.Wrapf(err, "failed to restore backup from file `%s`", a.config.FileStoragePath)
	}
	return nil
}

// Run запускает web сервер.
func (a *App) Run() error {
	if restoreErr := a.restoreBackup(); restoreErr != nil {
		return errors.Wrap(restoreErr, "run app error")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	server := controllers.SetupRouter(a.dbServices.URLService, a.config)
	var serverErr error
	go func() {
		defer func() {
			// если чан еще открыт, его нужно закрыть, чтоб приложение завершило свою работу.
			_, isOpen := <-quit
			if isOpen {
				close(quit)
			}
		}()
		if err := server.Run(a.config.ServerAddress); err != nil {
			a.config.Logger.WithError(err).Error("server error")
			// выставляем ошибку, чтоб вернуть её в main
			serverErr = err
		}
	}()

	<-quit
	// Делаем бекап
	a.config.Logger.Infof("Making backup to file `%s`", a.config.FileStoragePath)
	if backupErr := a.dbServices.URLService.Backup(a.config.FileStoragePath); backupErr != nil {
		a.config.Logger.WithError(backupErr).Error("backup error")
	} else {
		a.config.Logger.Info("Backup done")
	}

	// ошибка заполняется внутри горутины запуска сервера.
	if serverErr != nil {
		return serverErr
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
