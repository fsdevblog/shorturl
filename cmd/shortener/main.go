package main

import (
	"context"
	"errors"

	"github.com/fsdevblog/shorturl/internal/app"
	"github.com/fsdevblog/shorturl/internal/config"
)

func main() {
	appConf := config.MustLoadConfig()

	a := app.Must(app.New(*appConf))

	a.Logger.Debugf("Starting server with config %+v", appConf)
	if err := a.Run(); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}
