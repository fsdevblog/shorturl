package main

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/fsdevblog/shorturl/internal/app"
	"github.com/fsdevblog/shorturl/internal/config"
)

func main() {
	appConf := config.MustLoadConfig()

	a := app.Must(app.New(*appConf))

	a.Logger.Info("Starting server", zap.Any("config", appConf))
	if err := a.Run(); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}
