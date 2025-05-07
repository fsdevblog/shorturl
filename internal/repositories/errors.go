package repositories

import "errors"

var (
	ErrNotFound     = errors.New("[repository]: record not found")
	ErrDuplicateKey = errors.New("[repository]: duplicate key")
	ErrUnknown      = errors.New("[repository]: unknown error")
)
