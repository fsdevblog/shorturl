package main

import (
	"github.com/fsdevblog/shorturl/internal/app"
	"github.com/fsdevblog/shorturl/internal/config"
)

func main() {
	// Конфиг также можно перенести в app, но мне кажется ему все же место main, т.к. конфиг может
	// в будущем пригодится здесь.
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
