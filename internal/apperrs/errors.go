package apperrs

import "fmt"

type AppError struct {
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

var (
	ErrRecordNotFound = &AppError{
		Code:    "RECORD_NOT_FOUND",
		Message: "Record was not found",
	}
	ErrInternal = &AppError{
		Code:    "INTERNAL_ERROR",
		Message: "Internal server error",
	}
)
