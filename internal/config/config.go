package config

import (
	"flag"
	"fmt"
	"net/url"

	"github.com/caarlos0/env/v11"
)

type DBType string

const (
	DBTypeSQLite   DBType = "sqlite"
	DBTypeInMemory DBType = "inMemory"
)

type Config struct {
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	// Порт на котором запустится сервер
	ServerAddress string `env:"SERVER_ADDRESS"`
	// Базовый адрес результирующего сокращенного URL
	BaseURL *url.URL `env:"BASE_URL"`
	// Тип хранилища
	DBType DBType `env:"DB" envDefault:"inMemory"` // через флаги не настраиваю, незачем
}

func LoadConfig() (*Config, error) {
	var flagsConfig, envConfig Config

	if err := env.Parse(&envConfig); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	loadsFlags(&flagsConfig)

	conf := mergeConfig(&envConfig, &flagsConfig)
	return conf, nil
}

// MustLoad возвращает панику если произошла ошибка.
// Сделал отдельным методом по аналогии с библиотекой go-rod.
func MustLoad() *Config {
	conf, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	return conf
}

// loadsFlags парсит флаги командной строки.
func loadsFlags(flagsConfig *Config) {
	flag.StringVar(&flagsConfig.ServerAddress, "a", "localhost:8080", "Адрес сервера")
	flag.StringVar(&flagsConfig.FileStoragePath, "f", "backup.json", "Путь до файла бекапа")

	bDesc := "Базовый адрес результирующего сокращенного URL (по умолчанию Scheme://Host запущенного сервера)"
	flag.Func("b", bDesc, func(rawURL string) error {
		parsedURL, err := url.ParseRequestURI(rawURL)
		if err != nil {
			return fmt.Errorf("parse url: %w", err)
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
		ServerAddress:   defaultIfBlank[string](envConfig.ServerAddress, flagsConfig.ServerAddress),
		BaseURL:         defaultIfBlank[*url.URL](envConfig.BaseURL, flagsConfig.BaseURL),
		DBType:          defaultIfBlank[DBType](envConfig.DBType, flagsConfig.DBType),
		FileStoragePath: defaultIfBlank[string](envConfig.FileStoragePath, flagsConfig.FileStoragePath),
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
