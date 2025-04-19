package main

import (
	"github.com/fsdevblog/shorturl/internal/app"
	"github.com/fsdevblog/shorturl/internal/config"
)

func main() {
	appConf, confErr := config.LoadConfig()
	if confErr != nil {
		panic(confErr)
	}

	a := app.Must(app.New(appConf))

	appConf.Logger.Debugf("Starting server with config %+v", *appConf)
	if err := a.Run(); err != nil {
		panic(err)
	}
}
