package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsdevblog/shorturl/internal/transport/grpc/itrcpt"
	pb "github.com/fsdevblog/shorturl/internal/transport/grpc/proto"
	"github.com/fsdevblog/shorturl/internal/transport/trnptf"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	appgrpc "github.com/fsdevblog/shorturl/internal/transport/grpc"
	apphttp "github.com/fsdevblog/shorturl/internal/transport/http"

	"github.com/fsdevblog/shorturl/internal/services/svccert"

	"go.uber.org/zap"

	"github.com/fsdevblog/shorturl/internal/logs"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/db"
	"github.com/fsdevblog/shorturl/internal/services"
)

// Таймауты по умолчанию.
const (
	defaultReadHeaderTimeout = 5 * time.Second // таймаут чтения заголовков, во избежание Slowloris Attack
	defaultBackupTimeout     = 5 * time.Second // таймаут создания бекапа
	defaultShutdownTimeout   = 5 * time.Second // таймаут graceful shutdown
)

// Options опции для App.
type Options struct {
	ReadHeaderTimeout time.Duration // таймаут чтения заголовков, во избежание Slowloris Attack
	BackupTimeout     time.Duration // таймаут создания бекапа
	ShutdownTimeout   time.Duration // таймаут graceful shutdown
}

// App представляет собой основной объект приложения.
type App struct {
	config     config.Config      // Конфигурация приложения
	dbServices *services.Services // Сервисный слой для работы с БД
	Logger     *zap.Logger        // Логгер приложения

	readHeaderTimeout time.Duration
	backupTimeout     time.Duration
	shutdownTimeout   time.Duration
}

// New создает новый экземпляр приложения.
//
// Параметры:
//   - config: конфигурация приложения
//   - opts: опции
//
// Возвращает:
//   - *App: экземпляр приложения
//   - error: ошибка инициализации
func New(config config.Config, opts ...func(*Options)) (*App, error) {
	logger, errLogger := logs.New()
	if errLogger != nil {
		return nil, fmt.Errorf("init logger: %s", errLogger.Error())
	}

	ctx := context.Background()
	dbServices, servicesErr := initServices(ctx, config)

	if servicesErr != nil {
		return nil, fmt.Errorf("init services: %w", servicesErr)
	}

	options := &Options{
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		BackupTimeout:     defaultBackupTimeout,
		ShutdownTimeout:   defaultShutdownTimeout,
	}
	for _, opt := range opts {
		opt(options)
	}
	app := &App{
		config:            config,
		dbServices:        dbServices,
		Logger:            logger,
		readHeaderTimeout: options.ReadHeaderTimeout,
		backupTimeout:     options.BackupTimeout,
		shutdownTimeout:   options.ShutdownTimeout,
	}

	return app, nil
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

	// работа с сигналами уже реализована в предыдущих комитах.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return a.startHTTPServer(gctx)
	})
	g.Go(func() error {
		return a.startGRPCServer(gctx)
	})

	var errServer error
	if errServer = g.Wait(); errServer != nil {
		if !errors.Is(errServer, http.ErrServerClosed) {
			a.Logger.Error("server error", zap.Error(errServer))
		}

		stop()
	}

	a.Logger.Info("Shutdown command received")
	errServer = errors.Join(errServer, ctx.Err())

	backupCtx, backupCancel := context.WithTimeout(context.Background(), a.backupTimeout)
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

// startGRPCServer запускает GRPC сервер и блокирует дальнейшую работу горутины.
func (a *App) startGRPCServer(ctx context.Context) error {
	uf := trnptf.New(a.config.EnableHTTPS, a.config.ServerAddress, a.config.BaseURL, a.dbServices.URLService)
	srv := grpc.NewServer(grpc.UnaryInterceptor(itrcpt.Visitor))
	handler := appgrpc.New(uf)
	pb.RegisterShortURLServer(srv, handler)

	lis, err := net.Listen("tcp", a.config.GRPCAddress)
	if err != nil {
		return fmt.Errorf("start GRPC server: %w", err)
	}

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	if errServe := srv.Serve(lis); errServe != nil {
		return fmt.Errorf("serve GRPC server: %w", errServe)
	}
	return nil
}

// startHTTPServer запускает HTTP сервер и блокирует дальнейшую работу горутины.
func (a *App) startHTTPServer(ctx context.Context) error {
	router := apphttp.SetupRouter(apphttp.RouterParams{
		URLProvider: a.dbServices.URLService,
		PingService: a.dbServices.PingService,
		AppConf:     &a.config,
		Logger:      a.Logger,
	})

	httpSrv := &http.Server{
		Addr:              a.config.ServerAddress,
		Handler:           router,
		ReadHeaderTimeout: a.readHeaderTimeout,
	}
	lis, errLis := net.Listen("tcp", a.config.ServerAddress)
	if errLis != nil {
		return fmt.Errorf("start http server: %w", errLis)
	}

	var shutdownErr error
	go func() {
		<-ctx.Done()
		shutDownCtx, shutDownCancel := context.WithTimeout(ctx, a.shutdownTimeout)
		defer shutDownCancel()
		shutdownErr = httpSrv.Shutdown(shutDownCtx)
	}()

	if a.config.EnableHTTPS {
		certService := svccert.New(func(o *svccert.Options) {
			o.CertFilePath = "certs/cert.pem"
			o.KeyFilePath = "certs/key.pem"
		})
		errGen := certService.GenerateAndSaveIfNeed()
		if errGen != nil {
			return fmt.Errorf("start http server: %w", errGen)
		}

		err := httpSrv.ServeTLS(lis, certService.CertFilePath(), certService.KeyFilePath())
		if err != nil {
			return fmt.Errorf("start http server: %w", err)
		}
		return nil
	}
	if err := httpSrv.Serve(lis); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}
	if shutdownErr != nil {
		return fmt.Errorf("start http server: %w", shutdownErr)
	}
	return nil
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
