package config

import (
	"flag"
	"net/url"

	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

type Config struct {
	// Порт на котором запустится сервер
	ServerAddress string `env:"SERVER_ADDRESS"`
	// Базовый адрес результирующего сокращенного URL
	BaseURL *url.URL `env:"BASE_URL"`
}

func LoadConfig() *Config {
	var flagsConfig, envConfig Config

	if err := env.Parse(&envConfig); err != nil {
		panic(err)
	}

	loadsFlags(&flagsConfig)

	return mergeConfig(&envConfig, &flagsConfig)
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
	}
}

func defaultIfBlank[T any](value T, defaultValue T) T {
	if v, ok := any(value).(string); ok && v == "" {
		return defaultValue
	}
	if v, ok := any(value).(*url.URL); ok && v == nil {
		return defaultValue
	}
	return value
}
