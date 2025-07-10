package services

import "errors"

// ErrUnknown возвращается при неизвестной ошибке.
// ErrRecordNotFound возвращается, когда запрашиваемая запись не существует.
// ErrDuplicateKey возвращается при попытке создать дублирующуюся запись.
var (
	ErrUnknown        = errors.New("[service]: unknown error")
	ErrRecordNotFound = errors.New("[service]: record not found")
	ErrDuplicateKey   = errors.New("[service]: duplicate key")
)
