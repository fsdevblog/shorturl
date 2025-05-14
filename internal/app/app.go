package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsdevblog/shorturl/internal/logs"
	"github.com/sirupsen/logrus"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/controllers"
	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/services"
)

type App struct {
	config     config.Config
	dbServices *services.Services
	Logger     *logrus.Logger
}

func New(config config.Config) (*App, error) {
	logger := logs.New(os.Stdout)

	ctx := context.Background()
	dbServices, servicesErr := initServices(ctx, config)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:mnd
	defer cancel()

	if err := a.dbServices.URLService.RestoreBackup(ctx, a.config.FileStoragePath); err != nil {
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

	server := controllers.SetupRouter(controllers.RouterParams{
		URLService:  a.dbServices.URLService,
		PingService: a.dbServices.PingService,
		AppConf:     a.config,
		Logger:      a.Logger,
	})

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

	backupCtx, backupCancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:mnd
	defer backupCancel()

	// Делаем бекап
	// Из ТЗ не ясно, стоит делать бекап при подключении к БД или нет (такой бекап не имеет никакого смысла)
	// Лучше трогать не буду, проходят тесты и слава богу.
	if backupErr := a.dbServices.URLService.Backup(backupCtx, a.config.FileStoragePath); backupErr != nil {
		a.Logger.WithError(backupErr).
			Errorf("Making backup to file `%s` error", a.config.FileStoragePath)
	} else {
		a.Logger.Infof("Successfully made backup to file `%s`", a.config.FileStoragePath)
	}

	return serverErr
}

// initServices создает подключение к базе данных и возвращает сервисный слой приложения.
func initServices(ctx context.Context, appConf config.Config) (*services.Services, error) {
	// Нужно определить тип хранилища

	dbConn, connErr := db.NewConnectionFactory(ctx, db.FactoryConfig{
		StorageType: whatIsDBStorageType(&appConf),
		PostgresParams: &db.PostgresParams{
			DSN: appConf.DatabaseDSN,
		},
	})
	if connErr != nil {
		return nil, connErr //nolint:wrapcheck
	}

	dbServices, dbServErr := services.Factory(dbConn, whatIsServiceType(&appConf))
	if dbServErr != nil {
		return nil, dbServErr //nolint:wrapcheck
	}
	return dbServices, nil
}

func whatIsDBStorageType(appConf *config.Config) db.StorageType {
	if appConf.DatabaseDSN != "" {
		return db.StorageTypePostgres
	}
	return db.StorageTypeInMemory
}

func whatIsServiceType(appConf *config.Config) services.ServiceType {
	if appConf.DatabaseDSN != "" {
		return services.ServiceTypePostgres
	}
	return services.ServiceTypeInMemory
}
