package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/fsdevblog/shorturl/internal/logs"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/controllers"
	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/services"
)

type App struct {
	config     config.Config
	dbServices *services.Services
	Logger     *zap.Logger
}

func New(config config.Config) (*App, error) {
	logger, errLogger := logs.New()
	if errLogger != nil {
		return nil, fmt.Errorf("init logger: %s", errLogger.Error())
	}

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

	var errServer error
	select {
	case <-ctx.Done():
		a.Logger.Info("Shutdown command received")
		errServer = ctx.Err()
	case errServer = <-errChan:
		a.Logger.Error("router error", zap.Error(errServer))
	}

	backupCtx, backupCancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:mnd
	defer backupCancel()

	// Делаем бекап
	// Из ТЗ не ясно, стоит делать бекап при подключении к БД или нет (такой бекап не имеет никакого смысла)
	// Лучше трогать не буду, проходят тесты и слава богу.
	if errBackup := a.dbServices.URLService.Backup(backupCtx, a.config.FileStoragePath); errBackup != nil {
		a.Logger.Error("Making backup to file error",
			zap.String("file", a.config.FileStoragePath),
			zap.Error(errBackup),
		)
	} else {
		a.Logger.Info("Successfully made backup to file",
			zap.String("file", a.config.FileStoragePath),
		)
	}

	return errServer
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
