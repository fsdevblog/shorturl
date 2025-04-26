package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsdevblog/shorturl/internal/logs"
	"github.com/sirupsen/logrus"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/controllers"
	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/services"
)

type App struct {
	config     *config.Config
	dbServices *services.Services
	Logger     *logrus.Logger
}

func New(config *config.Config) (*App, error) {
	logger := logs.New(os.Stdout)
	dbServices, servicesErr := initServices(config, logger)

	if servicesErr != nil {
		return nil, fmt.Errorf("init services: %w", servicesErr)
	}

	return &App{
		config:     config,
		dbServices: dbServices,
		Logger:     logger,
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
		return fmt.Errorf("restore backup from file `%s`: %w", a.config.FileStoragePath, err)
	}
	return nil
}

// Run запускает web сервер.
func (a *App) Run() error {
	if restoreErr := a.restoreBackup(); restoreErr != nil {
		return fmt.Errorf("run app: %w", restoreErr)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 1)
	server := controllers.SetupRouter(a.dbServices.URLService, a.config, a.Logger)
	go func() {
		if err := server.Run(a.config.ServerAddress); err != nil {
			errChan <- err
		}
	}()

	var serverErr error
	select {
	case <-ctx.Done():
		a.Logger.Info("Shutdown command received")
	case serverErr = <-errChan:
		a.Logger.WithError(serverErr).Error("router error")
	}

	// Делаем бекап
	if backupErr := a.dbServices.URLService.Backup(a.config.FileStoragePath); backupErr != nil {
		a.Logger.WithError(backupErr).
			Errorf("Making backup to file `%s` error", a.config.FileStoragePath)
	} else {
		a.Logger.Infof("Successfully made backup to file `%s`", a.config.FileStoragePath)
	}

	return serverErr
}

// initServices создает подключение к базе данных и возвращает сервисный слой приложения.
func initServices(appConf *config.Config, logger *logrus.Logger) (*services.Services, error) {
	dbConn, connErr := db.NewConnection(db.StorageType(appConf.DBType))
	if connErr != nil {
		return nil, connErr //nolint:wrapcheck
	}

	dbServices, dbServErr := services.Factory(dbConn, services.ServiceType(appConf.DBType), logger)
	if dbServErr != nil {
		return nil, dbServErr //nolint:wrapcheck
	}
	return dbServices, nil
}
