package controllers

import "errors"

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrInternal       = errors.New("internal error")
)
