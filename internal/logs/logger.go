package logs

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EncodingType определяет формат вывода логов.
type EncodingType string

// LevelType определяет уровень логирования.
type LevelType string

// EncodingTypeConsole Форматирование для консоли.
// EncodingTypeJSON Форматирование в JSON.
const (
	EncodingTypeConsole EncodingType = "console"
	EncodingTypeJSON    EncodingType = "json"
)

// LevelTypeDebug Отладочный уровень.
// LevelTypeInfo Информационный уровень.
// LevelTypeWarning Уровень предупреждений.
// LevelTypeError Уровень ошибок.
// LevelTypeFatal Фатальный уровень.
// LevelTypePanic Уровень паники.
const (
	LevelTypeDebug   LevelType = "debug"
	LevelTypeInfo    LevelType = "info"
	LevelTypeWarning LevelType = "warning"
	LevelTypeError   LevelType = "error"
	LevelTypeFatal   LevelType = "fatal"
	LevelTypePanic   LevelType = "panic"
)

// LoggerOptions настройки логгера.
type LoggerOptions struct {
	Level            LevelType      // Уровень логирования
	Encoding         EncodingType   // Формат вывода
	OutputPaths      []string       // Пути вывода логов
	ErrorOutputPaths []string       // Пути вывода ошибок
	InitialFields    map[string]any // Начальные поля для каждой записи
}

// New создает новый логгер с указанными настройками.
//
// Параметры:
//   - opts: функции для настройки логгера
//
// Возвращает:
//   - *zap.Logger: настроенный логгер
//   - error: ошибка создания логгера
func New(opts ...func(*LoggerOptions)) (*zap.Logger, error) {
	isProduction := os.Getenv("GIN_RELEASE") == "release"

	var encoding = EncodingTypeConsole
	var level = LevelTypeDebug
	if isProduction {
		encoding = EncodingTypeJSON
		level = LevelTypeInfo
	}

	options := LoggerOptions{
		Level:            level,
		Encoding:         encoding,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	for _, opt := range opts {
		opt(&options)
	}

	lvl, errLvl := zap.ParseAtomicLevel(string(options.Level))
	if errLvl != nil {
		return nil, fmt.Errorf("parse level: %s", errLvl.Error())
	}

	conf := zap.Config{
		Level:             lvl,
		Development:       !isProduction,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          string(options.Encoding),
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:          "msg",
			LevelKey:            "level",
			TimeKey:             "ts",
			NameKey:             "logger",
			CallerKey:           "caller",
			FunctionKey:         zapcore.OmitKey,
			StacktraceKey:       "stacktrace",
			SkipLineEnding:      false,
			LineEnding:          zapcore.DefaultLineEnding,
			EncodeLevel:         zapcore.LowercaseLevelEncoder,
			EncodeTime:          zapcore.ISO8601TimeEncoder,
			EncodeDuration:      zapcore.StringDurationEncoder,
			EncodeCaller:        zapcore.ShortCallerEncoder,
			EncodeName:          nil,
			NewReflectedEncoder: nil,
			ConsoleSeparator:    "",
		},
		OutputPaths:      options.OutputPaths,
		ErrorOutputPaths: options.ErrorOutputPaths,
		InitialFields:    options.InitialFields,
	}

	log, err := conf.Build(zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		return nil, fmt.Errorf("build logger: %s", err.Error())
	}
	return log, nil
}

// MustNew создает новый логгер с указанными настройками.
// В случае ошибки вызывает panic.
//
// Параметры:
//   - opts: функции для настройки логгера
//
// Возвращает:
//   - *zap.Logger: настроенный логгер
func MustNew(opts ...func(*LoggerOptions)) *zap.Logger {
	log, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return log
}
