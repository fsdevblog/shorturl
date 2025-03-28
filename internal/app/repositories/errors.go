package repositories

import "errors"

var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key")
	ErrTransaction  = errors.New("transaction error")
	ErrUnknown      = errors.New("unknown error")
)
