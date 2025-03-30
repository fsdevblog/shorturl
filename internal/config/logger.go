package config

import (
	"os"

	"github.com/sirupsen/logrus"
)

// initLogger инициализирует логгер. В будущем сделаем его настраиваемым.
func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	logger.SetFormatter(new(logrus.JSONFormatter))
	logger.SetLevel(logrus.InfoLevel)

	// перезаписываем ряд настроек для окружений отличных от продакшн
	if os.Getenv("GIN_MODE") != "release" {
		logger.SetLevel(logrus.DebugLevel)
		logger.SetFormatter(new(logrus.TextFormatter))
	}

	return logger
}
