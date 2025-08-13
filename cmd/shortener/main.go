package main

import (
	"context"
	"errors"
	"fmt"

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

const defaultBuildMeta = "N/A"

func main() {
	appConf := config.MustLoadConfig()

	a := app.Must(app.New(*appConf))

	printBuildMeta()
	a.Logger.Info("Starting server", zap.Any("config", appConf))
	if err := a.Run(); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}

func printBuildMeta() {
	meta := struct {
		version string
		date    string
		commit  string
	}{
		version: defaultBuildMeta,
		date:    defaultBuildMeta,
		commit:  defaultBuildMeta,
	}
	if buildVersion != "" {
		meta.version = buildVersion
	}
	if buildDate != "" {
		meta.date = buildDate
	}
	if buildCommit != "" {
		meta.commit = buildCommit
	}

	fmt.Printf("Build version: %s\n", meta.version) //nolint:forbidigo
	fmt.Printf("Build date: %s\n", meta.date)       //nolint:forbidigo
	fmt.Printf("Build commit: %s\n", meta.commit)   //nolint:forbidigo
}
