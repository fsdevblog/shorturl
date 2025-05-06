package memory

import "errors"

var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key")
	ErrSerialize    = errors.New("serialize error")
)
