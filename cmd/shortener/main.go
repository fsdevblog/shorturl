package main

import (
	"context"
	"errors"

	"github.com/fsdevblog/shorturl/internal/bmeta"

	"go.uber.org/zap"

	"github.com/fsdevblog/shorturl/internal/app"
	"github.com/fsdevblog/shorturl/internal/config"
)

// nolint:gochecknoglobals
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	appConf := config.MustLoadConfig()

	a := app.Must(app.New(*appConf))

	bmeta.Print(buildVersion, buildDate, buildCommit)
	a.Logger.Info("Starting server", zap.Any("config", appConf))
	if err := a.Run(); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}
