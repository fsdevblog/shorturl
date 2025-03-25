package main

import (
	"os"

	"github.com/fsdevblog/shorturl/internal/app/repositories/sqlite"
	"github.com/fsdevblog/shorturl/internal/app/services"

	"github.com/fsdevblog/shorturl/internal/app/db"
	"github.com/fsdevblog/shorturl/internal/app/server"
	"github.com/sirupsen/logrus"
)

// DBPath Потом перенесу в .env.
const DBPath = "./shortener.sqlite"

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{})

	dbConn := db.ConnectSQLite(DBPath)
	db.MigrateSQLite(dbConn)

	urlRepo := sqlite.NewURLRepo(dbConn)

	urlService := services.NewURLService(urlRepo)

	server.Start(urlService)
}
