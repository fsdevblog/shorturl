package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/caarlos0/env/v11"
)

// Config содержит параметры конфигурации приложения.
type Config struct {
	// Путь для бекапа (актуально мемори хранилища).
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	// HTTPS сервер.
	EnableHTTPS bool `env:"ENABLE_HTTPS" envDefault:"false" json:"enable_https"`
	// Конфиг файл.
	ConfigJSON string `env:"CONFIG" json:"-"`
	// CIDR
	TrustedSubnet string `env:"TRUSTED_SUBNET" json:"trusted_subnet"`
	// Адрес сервера.
	ServerAddress string `env:"SERVER_ADDRESS" json:"server_address"`
	// Базовый адрес результирующего сокращенного URL
	BaseURL string `env:"BASE_URL" json:"base_url"`
	// DSN базы данных
	DatabaseDSN string `env:"DATABASE_DSN" json:"database_dsn"`
	// Секретный ключ для JWT токена посетителей.
	VisitorJWTSecret string `env:"VISITOR_JWT_SECRET" envDefault:"super_secret_key" json:"-"`
	// Адрес GRPC сервера.
	GRPCAddress string `env:"GRPC_ADDRESS" json:"grpc_address"`
}

// readConfigFile читает и парсит файл конфигурации в структуру Config.
func readConfigFile(configFilePath string) (*Config, error) {
	cfgBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return parseConfigFile(cfgBytes)
}

// parseConfigFile парсит файл конфигурации в структуру Config.
func parseConfigFile(cfgBytes []byte) (*Config, error) {
	var conf Config
	errUnmarshal := json.Unmarshal(cfgBytes, &conf)
	if errUnmarshal != nil {
		return nil, fmt.Errorf("unmarshal config: %w", errUnmarshal)
	}
	return &conf, nil
}

// LoadConfig загружает конфигурацию из переменных окружения и флагов командной строки.
// Приоритет: переменные окружения > флаги командной строки > значения по умолчанию.
//
// Поддерживаемые переменные окружения:
//   - FILE_STORAGE_PATH: путь к файлу хранилища
//   - ENABLE_HTTPS: запуск HTTPS сервера (true/false)
//   - CONFIG: имя файла конфигурации
//   - TRUSTED_SUBNET: CIDR
//   - SERVER_ADDRESS: адрес сервера
//   - BASE_URL: базовый URL для сокращенных ссылок
//   - DATABASE_DSN: строка подключения к БД
//   - VISITOR_JWT_SECRET: секрет для JWT (по умолчанию "super_secret_key")
//   - GRPC_ADDRESS: адрес GRPC сервера (по умолчанию ":50051")
//
// Поддерживаемые флаги:
//   - -f: путь к файлу хранилища (по умолчанию "backup.json")
//   - -s: запуск HTTPS сервера (true/false)
//   - -c: имя файла конфигурации
//   - -t: CIDR
//   - -a: адрес сервера (по умолчанию "localhost:8080")
//   - -d: строка подключения к БД
//   - -b: базовый URL для сокращенных ссылок
//   - -g: адрес GRPC сервера
//
// Возвращает:
//   - *Config: загруженная конфигурация
//   - error: ошибка загрузки конфигурации
func LoadConfig() (*Config, error) {
	var flagsConfig, envConfig Config

	if err := env.Parse(&envConfig); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	loadsFlags(&flagsConfig)

	configFile := flagsConfig.ConfigJSON
	if configFile == "" {
		configFile = envConfig.ConfigJSON
	}

	fileConfig, err := readConfigFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	conf := mergeConfigs(&envConfig, &flagsConfig, fileConfig)

	return conf, nil
}

// mergeConfigs объединяет конфигурации из всех источников с учетом приоритетов.
// Порядок приоритетов (от высшего к низшему):
// 1. Флаги командной строки
// 2. Переменные окружения
// 3. Файл конфигурации.
func mergeConfigs(f, e, fl *Config) *Config {
	return &Config{
		ServerAddress:    firstNonEmpty(f.ServerAddress, e.ServerAddress, fl.ServerAddress),
		BaseURL:          firstNonEmpty(f.BaseURL, e.BaseURL, fl.BaseURL),
		DatabaseDSN:      firstNonEmpty(f.DatabaseDSN, e.DatabaseDSN, fl.DatabaseDSN),
		TrustedSubnet:    firstNonEmpty(f.TrustedSubnet, e.TrustedSubnet, fl.TrustedSubnet),
		FileStoragePath:  firstNonEmpty(f.FileStoragePath, e.FileStoragePath, fl.FileStoragePath),
		EnableHTTPS:      f.EnableHTTPS || e.EnableHTTPS || fl.EnableHTTPS,
		VisitorJWTSecret: firstNonEmpty(f.VisitorJWTSecret, e.VisitorJWTSecret, fl.VisitorJWTSecret),
		GRPCAddress:      firstNonEmpty(f.GRPCAddress, e.GRPCAddress, fl.GRPCAddress),
	}
}

// firstNonEmpty возвращает первое непустое значение из списка или значение типа T по умолчанию.
func firstNonEmpty[T comparable](values ...T) T {
	var zero T
	for _, v := range values {
		if v != zero {
			return v
		}
	}
	return zero
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
//   - -c: имя файла конфигурации
//   - -t: CIDR
//   - -f: путь к файлу хранилища (по умолчанию "backup.json")
//   - -d: строка подключения к БД
//   - -b: базовый URL для сокращенных ссылок (scheme://host)
//   - -g: адрес GRPC сервера
//
// Параметры:
//   - flagsConfig: указатель на структуру для сохранения значений флагов
func loadsFlags(flagsConfig *Config) {
	flag.StringVar(&flagsConfig.ServerAddress, "a", "localhost:8080", "Адрес сервера")
	flag.BoolVar(&flagsConfig.EnableHTTPS, "s", false, "Запуск HTTPS")
	flag.StringVar(&flagsConfig.ConfigJSON, "c", "config.json", "Имя файла конфигурации")
	flag.StringVar(&flagsConfig.TrustedSubnet, "t", "", "CIDR")
	flag.StringVar(&flagsConfig.FileStoragePath, "f", "backup.json", "Путь до файла бекапа")
	flag.StringVar(&flagsConfig.DatabaseDSN, "d", "", "DSN подключения к СУБД")
	flag.StringVar(&flagsConfig.GRPCAddress, "g", ":50051", "адрес GRPC сервера")

	bDesc := "Базовый адрес результирующего сокращенного URL (по умолчанию Scheme://Host запущенного сервера)"
	flag.Func("b", bDesc, func(rawURL string) error {
		parsedURL, err := url.ParseRequestURI(rawURL)
		if err != nil {
			return fmt.Errorf("parse base url: %w", err)
		}

		s := url.URL{
			Scheme: parsedURL.Scheme,
			Host:   parsedURL.Host,
		}
		// создаем новый инстанс, отсекая тем самым Path и Query если они заданы в базовом урле.
		flagsConfig.BaseURL = s.String()
		return nil
	})

	flag.Parse()
}
