package logs

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// New инициализирует логгер.
func New(output io.Writer) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(output)

	logger.SetFormatter(new(logrus.JSONFormatter))
	logger.SetLevel(logrus.InfoLevel)

	// перезаписываем ряд настроек для окружений отличных от продакшн
	if os.Getenv("GIN_MODE") != "release" {
		logger.SetLevel(logrus.DebugLevel)
		logger.SetFormatter(new(logrus.TextFormatter))
	}

	return logger
}
