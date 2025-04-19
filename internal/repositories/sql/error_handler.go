package sql

import (
	"github.com/fsdevblog/shorturl/internal/repositories"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func ConvertErrorType(err error) error {
	switch {
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return repositories.ErrDuplicateKey
	case errors.Is(err, gorm.ErrRecordNotFound):
		return repositories.ErrNotFound
	default:
		return repositories.ErrUnknown
	}
}
