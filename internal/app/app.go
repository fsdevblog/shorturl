package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsdevblog/shorturl/internal/services/svccert"

	"go.uber.org/zap"

	"github.com/fsdevblog/shorturl/internal/logs"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/controllers"
	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/services"
)

// App представляет собой основной объект приложения.
type App struct {
	config     config.Config      // Конфигурация приложения
	dbServices *services.Services // Сервисный слой для работы с БД
	Logger     *zap.Logger        // Логгер приложения
}

// New создает новый экземпляр приложения.
//
// Параметры:
//   - config: конфигурация приложения
//
// Возвращает:
//   - *App: экземпляр приложения
//   - error: ошибка инициализации
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

// Must обертка над конструктором, вызывающая panic при ошибке.
//
// Параметры:
//   - a: экземпляр приложения
//   - err: ошибка
//
// Возвращает:
//   - *App: экземпляр приложения
//
// Паникует при err != nil.
func Must(a *App, err error) *App {
	if err != nil {
		panic(err)
	}
	return a
}

// restoreBackup восстанавливает данные из резервной копии (для in-memory хранилища).
// Использует таймаут 10 секунд для операции восстановления.
//
// Возвращает:
//   - error: ошибка восстановления данных из файла
func (a *App) restoreBackup() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:mnd
	defer cancel()

	if err := a.dbServices.URLService.RestoreBackup(ctx, a.config.FileStoragePath); err != nil {
		return fmt.Errorf("restore backup from file `%s`: %w", a.config.FileStoragePath, err)
	}
	return nil
}

// Run запускает web сервер и обрабатывает сигналы завершения.
// При получении сигнала SIGINT или SIGTERM выполняет корректное завершение:
//   - Создает резервную копию данных, если используется in-memory хранилище.
//   - Завершает работу сервера.
//
// Возвращает:
//   - error: ошибка работы сервера
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
		if a.config.EnableHTTPS {
			certService := svccert.New()
			errGen := certService.GenerateAndSaveIfNeed()
			if errGen != nil {
				errChan <- errGen
				return
			}

			cert, key, errRead := certService.PairString()
			if errRead != nil {
				errChan <- errRead
				return
			}

			err := server.RunTLS(a.config.ServerAddress, cert, key)
			if err != nil {
				errChan <- err
			}
			return
		}

		err := server.Run(a.config.ServerAddress)
		if err != nil {
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

// initServices инициализирует сервисный слой приложения.
// Определяет тип хранилища (PostgreSQL или in-memory) на основе конфигурации.
//
// Параметры:
//   - ctx: контекст выполнения
//   - appConf: конфигурация приложения
//
// Возвращает:
//   - *services.Services: инициализированный сервисный слой
//   - error: ошибка инициализации
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

// whatIsDBStorageType определяет тип хранилища на основе конфигурации.
//
// Параметры:
//   - appConf: конфигурация приложения
//
// Возвращает:
//   - db.StorageType: тип хранилища (StorageTypePostgres или StorageTypeInMemory)
func whatIsDBStorageType(appConf *config.Config) db.StorageType {
	if appConf.DatabaseDSN != "" {
		return db.StorageTypePostgres
	}
	return db.StorageTypeInMemory
}

// whatIsServiceType определяет тип сервиса на основе конфигурации.
//
// Параметры:
//   - appConf: конфигурация приложения
//
// Возвращает:
//   - services.ServiceType: тип сервиса (ServiceTypePostgres или ServiceTypeInMemory)
func whatIsServiceType(appConf *config.Config) services.ServiceType {
	if appConf.DatabaseDSN != "" {
		return services.ServiceTypePostgres
	}
	return services.ServiceTypeInMemory
}
