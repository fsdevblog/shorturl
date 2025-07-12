package controllers

import "errors"

// Ошибки.
var (
	ErrRecordNotFound = errors.New("record not found") // Запись не найдена
	ErrInternal       = errors.New("internal error")   // Прочая ошибка
)
