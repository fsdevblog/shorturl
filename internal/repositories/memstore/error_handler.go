package memstore

import (
	"errors"
	"fmt"

	"github.com/fsdevblog/shorturl/internal/db/memory"
	"github.com/fsdevblog/shorturl/internal/repositories"
)

// convertErrorType конвертирует специфичные ошибки хранилища в памяти
// в общие ошибки уровня репозитория.
//
// Параметры:
//   - err: исходная ошибка
//
// Возвращает:
//   - error: преобразованная ошибка или nil, если входная ошибка nil
//
// Преобразования ошибок:
//   - memory.ErrDuplicateKey -> repositories.ErrDuplicateKey
//   - memory.ErrNotFound -> repositories.ErrNotFound
//   - другие ошибки -> repositories.ErrUnknown
func convertErrorType(err error) error {
	if err == nil {
		return nil
	}

	var nativeErr error
	switch {
	case errors.Is(err, memory.ErrDuplicateKey):
		nativeErr = repositories.ErrDuplicateKey
	case errors.Is(err, memory.ErrNotFound):

		nativeErr = repositories.ErrNotFound
	default:
		nativeErr = repositories.ErrUnknown
	}

	return fmt.Errorf("%w: %s", nativeErr, err.Error())
}
