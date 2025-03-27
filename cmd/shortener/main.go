package main

import (
	"os"

	"github.com/fsdevblog/shorturl/internal/app/config"

	"github.com/fsdevblog/shorturl/internal/app/controllers"

	"github.com/fsdevblog/shorturl/internal/app/repositories/sqlite"
	"github.com/fsdevblog/shorturl/internal/app/services"

	"github.com/fsdevblog/shorturl/internal/app/db"
	"github.com/sirupsen/logrus"
)

// DBPath Потом перенесу в .env.
const DBPath = "./shortener.sqlite"

func main() {
	logLevel := logrus.DebugLevel
	if os.Getenv("GIN_MODE") == "release" {
		logLevel = logrus.InfoLevel
	}

	logrus.SetLevel(logLevel)
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{})

	dbConn := db.ConnectSQLite(DBPath)
	db.MigrateSQLite(dbConn)

	urlRepo := sqlite.NewURLRepo(dbConn)

	urlService := services.NewURLService(urlRepo)

	appConf := config.LoadConfig()
	router := controllers.SetupRouter(urlService, appConf)

	logrus.Debugf("Server is running with config: %+v", *appConf)
	routerErr := router.Run(appConf.ServerAddress)
	if routerErr != nil {
		logrus.Fatal(routerErr)
	}
}
