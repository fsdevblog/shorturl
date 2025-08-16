package config

import (
	"flag"
	"fmt"
	"net/url"

	"github.com/caarlos0/env/v11"
)

// Config содержит параметры конфигурации приложения.
type Config struct {
	// Путь для бекапа (актуально мемори хранилища).
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	// HTTPS сервер.
	EnableHTTPS bool `env:"ENABLE_HTTPS" envDefault:"false"`
	// Порт на котором запустится сервер
	ServerAddress string `env:"SERVER_ADDRESS"`
	// Базовый адрес результирующего сокращенного URL
	BaseURL *url.URL `env:"BASE_URL"`
	// DSN базы данных
	DatabaseDSN string `env:"DATABASE_DSN"`
	// Секретный ключ для JWT токена посетителей.
	VisitorJWTSecret string `env:"VISITOR_JWT_SECRET" envDefault:"super_secret_key"`
}

// LoadConfig загружает конфигурацию из переменных окружения и флагов командной строки.
// Приоритет: переменные окружения > флаги командной строки > значения по умолчанию.
//
// Поддерживаемые переменные окружения:
//   - FILE_STORAGE_PATH: путь к файлу хранилища
//   - ENABLE_HTTPS: запуск HTTPS сервера (true/false)
//   - SERVER_ADDRESS: адрес сервера
//   - BASE_URL: базовый URL для сокращенных ссылок
//   - DATABASE_DSN: строка подключения к БД
//   - VISITOR_JWT_SECRET: секрет для JWT (по умолчанию "super_secret_key")
//
// Поддерживаемые флаги:
//   - -f: путь к файлу хранилища (по умолчанию "backup.json")
//   - -s: запуск HTTPS сервера (true/false)
//   - -a: адрес сервера (по умолчанию "localhost:8080")
//   - -d: строка подключения к БД
//   - -b: базовый URL для сокращенных ссылок
//
// Возвращает:
//   - *Config: загруженная конфигурация
//   - error: ошибка загрузки конфигурации
func LoadConfig() (*Config, error) {
	var flagsConfig, envConfig Config

	if err := env.Parse(&envConfig); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	loadsFlags(&flagsConfig)

	conf := mergeConfig(&envConfig, &flagsConfig)
	return conf, nil
}

// MustLoadConfig аналогичен LoadConfig, но вызывает panic при ошибке.
//
// Возвращает:
//   - *Config: загруженная конфигурация
//
// Паникует при ошибке загрузки конфигурации.
func MustLoadConfig() *Config {
	conf, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	return conf
}

// loadsFlags парсит флаги командной строки в структуру Config.
//
// Поддерживаемые флаги:
//   - -a: адрес сервера (по умолчанию "localhost:8080")
//   - -s: запуск HTTPS сервера (true/false)
//   - -f: путь к файлу хранилища (по умолчанию "backup.json")
//   - -d: строка подключения к БД
//   - -b: базовый URL для сокращенных ссылок (scheme://host)
//
// Параметры:
//   - flagsConfig: указатель на структуру для сохранения значений флагов
func loadsFlags(flagsConfig *Config) {
	flag.StringVar(&flagsConfig.ServerAddress, "a", "localhost:8080", "Адрес сервера")
	flag.BoolVar(&flagsConfig.EnableHTTPS, "s", false, "Запуск HTTPS")
	flag.StringVar(&flagsConfig.FileStoragePath, "f", "backup.json", "Путь до файла бекапа")
	flag.StringVar(&flagsConfig.DatabaseDSN, "d", "", "DSN подключения к СУБД")

	bDesc := "Базовый адрес результирующего сокращенного URL (по умолчанию Scheme://Host запущенного сервера)"
	flag.Func("b", bDesc, func(rawURL string) error {
		parsedURL, err := url.ParseRequestURI(rawURL)
		if err != nil {
			return fmt.Errorf("parse base url: %w", err)
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

// mergeConfig объединяет конфигурации из переменных окружения и флагов.
// Приоритет отдается значениям из переменных окружения.
//
// Параметры:
//   - envConfig: конфигурация из переменных окружения
//   - flagsConfig: конфигурация из флагов командной строки
//
// Возвращает:
//   - *Config: объединенная конфигурация
func mergeConfig(envConfig, flagsConfig *Config) *Config {
	return &Config{
		ServerAddress:    defaultIfBlank[string](envConfig.ServerAddress, flagsConfig.ServerAddress),
		BaseURL:          defaultIfBlank[*url.URL](envConfig.BaseURL, flagsConfig.BaseURL),
		DatabaseDSN:      defaultIfBlank[string](envConfig.DatabaseDSN, flagsConfig.DatabaseDSN),
		FileStoragePath:  defaultIfBlank[string](envConfig.FileStoragePath, flagsConfig.FileStoragePath),
		EnableHTTPS:      defaultIfBlank[bool](envConfig.EnableHTTPS, flagsConfig.EnableHTTPS),
		VisitorJWTSecret: envConfig.VisitorJWTSecret,
	}
}

// defaultIfBlank возвращает значение по умолчанию, если переданное значение пустое.
// Поддерживает типы string и *url.URL.
//
// Параметры:
//   - value: проверяемое значение
//   - defaultValue: значение по умолчанию
//
// Возвращает:
//   - T: исходное значение или значение по умолчанию
func defaultIfBlank[T string | *url.URL | bool](value T, defaultValue T) T {
	if v, ok := any(value).(string); ok && v == "" {
		return defaultValue
	}
	if v, ok := any(value).(*url.URL); ok && v == nil {
		return defaultValue
	}
	if v, ok := any(value).(bool); ok && !v {
		return defaultValue
	}
	return value
}
