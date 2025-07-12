package tokens

import "errors"

// ErrTokenExpired токен просрочен.
var (
	ErrTokenExpired = errors.New("token expired")
)
