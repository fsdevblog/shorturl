package http

import (
	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/transport/http/middlewares"
	"github.com/fsdevblog/shorturl/internal/transport/trnptf"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RouterParams определяет параметры для настройки маршрутизатора.
type RouterParams struct {
	URLProvider  trnptf.URLProvider       // Сервис для работы с короткими URL
	StatsService trnptf.StatsProvider     // Сервис статистики
	PingService  trnptf.ConnectionChecker // Сервис для проверки работоспособности системы
	AppConf      *config.Config           // Конфигурация приложения
	Logger       *zap.Logger              // Логгер приложения
}

// SetupRouter настраивает и возвращает маршрутизатор приложения.
// Регистрирует все обработчики HTTP запросов и промежуточные обработчики (middleware).
//
// Регистрируемые middleware:
//   - gin.Recovery() для восстановления после паник
//   - LoggerMiddleware для логирования запросов (если Logger != nil)
//   - pprof для профилирования
//   - VisitorCookieMiddleware для идентификации пользователей
//   - GzipMiddleware для сжатия ответов
//
// Регистрируемые маршруты:
//
//	GET /:shortID - редирект по короткому URL
//	POST / - создание короткого URL
//	GET /ping - проверка работоспособности
//
// API маршруты (/api/...):
//
//	POST /shorten - создание короткого URL
//	POST /shorten/batch - пакетное создание коротких URL
//	GET /:shortID - редирект по короткому URL
//	GET /user/urls - получение URL пользователя
//	GET /internal/stats - статистика
//	DELETE /user/urls - удаление URL пользователя
//
// Параметры:
//   - params: параметры для настройки маршрутизатора
//
// Возвращает:
//   - *gin.Engine: настроенный маршрутизатор
func SetupRouter(params RouterParams) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	if params.Logger != nil {
		r.Use(middlewares.LoggerMiddleware(params.Logger))
	}

	// подключаем pprof. Т.к. задачи защищать роут в продакшн окружении не стоит, не делаем этого.
	pprof.Register(r)

	r.Use(middlewares.VisitorCookieMiddleware([]byte(params.AppConf.VisitorJWTSecret)))
	r.Use(middlewares.GzipMiddleware())

	urlFacade := trnptf.New(
		params.AppConf.EnableHTTPS,
		params.AppConf.ServerAddress,
		params.AppConf.BaseURL,
		params.URLProvider,
	)
	shortURLController := NewShortURLController(urlFacade)
	pingController := NewPingController(params.PingService)
	statsController := NewStatsController(params.StatsService)

	r.GET("/:shortID", shortURLController.Redirect)
	r.POST("/", shortURLController.CreateShortURL)
	r.GET("/ping", pingController.Ping)

	api := r.Group("/api")
	api.POST("/shorten", shortURLController.CreateShortURL)
	api.POST("/shorten/batch", shortURLController.BatchCreate)
	api.GET("/:shortID", shortURLController.Redirect)
	api.GET("/user/urls", shortURLController.UserURLs)
	api.DELETE("/user/urls", shortURLController.DeleteUserURLs)

	internal := api.Group("/internal")
	internal.Use(middlewares.InternalAccess(params.AppConf.TrustedSubnet))

	internal.GET("/stats", statsController.GetStats)
	return r
}
