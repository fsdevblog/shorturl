package repositories

import "errors"

// ErrNotFound возвращается, когда запрашиваемая запись не найдена в хранилище
// ErrDuplicateKey возвращается при попытке создать запись с уже существующим ключом
// ErrUnknown возвращается при неизвестной ошибке на уровне репозитория.
var (
	ErrNotFound     = errors.New("[repository]: record not found")
	ErrDuplicateKey = errors.New("[repository]: duplicate key")
	ErrUnknown      = errors.New("[repository]: unknown error")
)
