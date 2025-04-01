package repositories

import "errors"

var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key")
	ErrUnknown      = errors.New("unknown error")
)
