package sql

import (
	"errors"
	"fmt"

	"github.com/fsdevblog/shorturl/internal/repositories"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolationCode = "23505"
)

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

func isUniqueViolationErr(err *pgconn.PgError) bool {
	return err.Code == uniqueViolationCode
}
