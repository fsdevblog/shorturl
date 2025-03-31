package config

import (
	"flag"
	"net/url"

	"github.com/sirupsen/logrus"

	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

type DBType string

const (
	DBTypeSQLite   DBType = "sqlite"
	DBTypeInMemory DBType = "inMemory"
)

type Config struct {
	// Порт на котором запустится сервер
	ServerAddress string `env:"SERVER_ADDRESS"`
	// Базовый адрес результирующего сокращенного URL
	BaseURL *url.URL `env:"BASE_URL"`
	// Тип хранилища
	DBType DBType `env:"DB" envDefault:"inMemory"` // через флаги не настраиваю, незачем
	Logger *logrus.Logger
}

func LoadConfig() (*Config, error) {
	var flagsConfig, envConfig Config
	logger := initLogger()

	if err := env.Parse(&envConfig); err != nil {
		return nil, errors.Wrapf(err, "parse ENV config error")
	}

	loadsFlags(&flagsConfig)

	conf := mergeConfig(&envConfig, &flagsConfig)
	conf.Logger = logger
	return conf, nil
}

// loadsFlags парсит флаги командной строки.
func loadsFlags(flagsConfig *Config) {
	flag.StringVar(&flagsConfig.ServerAddress, "a", "localhost:8080", "Адрес сервера")

	bDesc := "Базовый адрес результирующего сокращенного URL (по умолчанию Scheme://Host запущенного сервера)"
	flag.Func("b", bDesc, func(rawURL string) error {
		parsedURL, err := url.ParseRequestURI(rawURL)
		if err != nil {
			return errors.Wrap(err, "failed to parse base url")
		}

		// создаем новый инстанс, отсекая тем самым Path и Query если они заданы в базовом урле.
		flagsConfig.BaseURL = &url.URL{
			Scheme: parsedURL.Scheme,
			Host:   parsedURL.Host,
		}
		return nil
	})

	flag.Parse()
}

// mergeConfig сливает структуры для env и флагов.
func mergeConfig(envConfig, flagsConfig *Config) *Config {
	return &Config{
		ServerAddress: defaultIfBlank[string](envConfig.ServerAddress, flagsConfig.ServerAddress),
		BaseURL:       defaultIfBlank[*url.URL](envConfig.BaseURL, flagsConfig.BaseURL),
		DBType:        defaultIfBlank[DBType](envConfig.DBType, flagsConfig.DBType),
	}
}

func defaultIfBlank[T any](value T, defaultValue T) T {
	if v, ok := any(value).(string); ok && v == "" {
		return defaultValue
	}
	if v, ok := any(value).(DBType); ok && v == "" {
		return defaultValue
	}
	if v, ok := any(value).(*url.URL); ok && v == nil {
		return defaultValue
	}
	return value
}
