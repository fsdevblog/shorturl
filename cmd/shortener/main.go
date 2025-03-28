package main

import (
	"github.com/fsdevblog/shorturl/internal/app"
	"github.com/fsdevblog/shorturl/internal/app/config"
	"github.com/fsdevblog/shorturl/internal/app/db"
	"github.com/fsdevblog/shorturl/internal/app/services"
)

func main() {
	appConf, confErr := config.LoadConfig()
	if confErr != nil {
		panic(confErr)
	}

	dbServices, dbErr := initDB(appConf)
	if dbErr != nil {
		panic(dbErr)
	}

	app.NewApp(appConf, dbServices).Run()
}

func initDB(appConf *config.Config) (*services.Services, error) {
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
