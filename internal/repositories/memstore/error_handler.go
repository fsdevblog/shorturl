package memstore

import (
	"errors"
	"fmt"

	"github.com/fsdevblog/shorturl/internal/db/memory"
	"github.com/fsdevblog/shorturl/internal/repositories"
)

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
