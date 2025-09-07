package trnptf

import (
	"errors"
	"fmt"
)

// InvalidURLInBatchError Ошибка, содержащая некорректный урл при пакетной вставке.
type InvalidURLInBatchError struct {
	CorrelationID string
	OriginalURL   string
}

// NewInvalidURLInBatch возвращает новый объект InvalidURLInBatchError.
func NewInvalidURLInBatch(correlationID, originalURL string) *InvalidURLInBatchError {
	return &InvalidURLInBatchError{
		CorrelationID: correlationID,
		OriginalURL:   originalURL,
	}
}

// Error приводит ошибку к строке.
func (i *InvalidURLInBatchError) Error() string {
	return fmt.Sprintf("URL %q is invalid", i.OriginalURL)
}

// Ошибки.
var (
	ErrInvalidURL    = errors.New("URL is invalid") // URL не валиден.
	ErrBadRequest    = errors.New("bad request")    // Некорректный запрос.
	ErrRecordDeleted = errors.New("record deleted") // Запись была удалена.
)
