package sql

import (
	"errors"
	"fmt"

	"github.com/fsdevblog/shorturl/internal/repositories"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolationCode = "23505" // Код ошибки нарушения уникальности
)

// convertErrType конвертирует ошибки PostgreSQL в общие ошибки уровня репозитория.
//
// Параметры:
//   - err: исходная ошибка
//
// Возвращает:
//   - error: преобразованная ошибка или nil, если входная ошибка nil
//
// Преобразования ошибок:
//   - uniqueViolationCode (23505) -> repositories.ErrDuplicateKey
//   - другие ошибки -> repositories.ErrUnknown
func convertErrType(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		errType := repositories.ErrUnknown
		// Потом гляну какие еще можно сюда типы ошибок прикрутить, но пока вроде хватает
		if isUniqueViolationErr(pgErr) {
			errType = repositories.ErrDuplicateKey
		}

		return fmt.Errorf("%w: %s", errType, pgErr.Message)
	}

	return fmt.Errorf("%w: %s", repositories.ErrUnknown, err.Error())
}

// isUniqueViolationErr проверяет, является ли ошибка нарушением уникальности.
//
// Параметры:
//   - err: ошибка PostgreSQL
//
// Возвращает:
//   - bool: true если это ошибка нарушения уникальности
func isUniqueViolationErr(err *pgconn.PgError) bool {
	return err.Code == uniqueViolationCode
}
