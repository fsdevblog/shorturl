package memory

import "errors"

// ErrNotFound возвращается, когда запрашиваемая запись не найдена в хранилище
// ErrDuplicateKey возвращается при попытке создать запись с уже существующим ключом
// ErrSerialize возвращается при ошибке сериализации данных.
var (
	ErrNotFound     = errors.New("[memory]: record not found")
	ErrDuplicateKey = errors.New("[memory]: duplicate Key")
	ErrSerialize    = errors.New("[memory]: serialize error")
)
