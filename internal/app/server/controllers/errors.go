package controllers

import "errors"

var (
	ErrBadRequest       = errors.New("bad request")
	ErrNotFound         = errors.New("not found")
	ErrMethodNowAllowed = errors.New("method not allowed")
	ErrInternal         = errors.New("internal server error")
)
