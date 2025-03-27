package config

import (
	"flag"
	"net/url"

	"github.com/pkg/errors"
)

type Config struct {
	// Порт на котором запустится сервер
	ServerAddress string
	// Базовый адрес результирующего сокращенного URL
	BaseURL *url.URL
}

func LoadConfig() *Config {
	var config Config
	bDesc := "Базовый адрес результирующего сокращенного URL (по умолчанию Scheme://Host запущенного сервера)"

	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "Адрес сервера")
	flag.Func("b", bDesc, func(rawURL string) error {
		parsedURL, err := url.ParseRequestURI(rawURL)
		if err != nil {
			return errors.Wrap(err, "failed to parse base url")
		}

		// создаем новый инстанс, отсекая тем самым Path и Query если они заданы в базовом урле.
		config.BaseURL = &url.URL{
			Scheme: parsedURL.Scheme,
			Host:   parsedURL.Host,
		}
		return nil
	})
	flag.Parse()
	return &config
}
