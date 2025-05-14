package memory

import "errors"

var (
	ErrNotFound     = errors.New("[memory]: record not found")
	ErrDuplicateKey = errors.New("[memory]: duplicate key")
	ErrSerialize    = errors.New("[memory]: serialize error")
)
