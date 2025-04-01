package services

import "errors"

var (
	ErrUnknown        = errors.New("unknown error")
	ErrRecordNotFound = errors.New("record not found")
)
