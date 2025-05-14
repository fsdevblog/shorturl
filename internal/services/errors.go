package services

import "errors"

var (
	ErrUnknown        = errors.New("[service]: unknown error")
	ErrRecordNotFound = errors.New("[service]: record not found")
	ErrDuplicateKey   = errors.New("[service]: duplicate key")
)
