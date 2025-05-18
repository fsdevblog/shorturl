package tokens

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

type VisitorClaims struct {
	jwt.RegisteredClaims
	UUID string
}

func GenerateVisitorJWT(uuid string, expire time.Duration, key []byte) (string, error) {
	visitorClaims := VisitorClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
		},
		UUID: uuid,
	}
	token, err := generateJWT(visitorClaims, key)
	if err != nil {
		return "", fmt.Errorf("generating visitor jwt token: %w", err)
	}
	return token, nil
}

func ValidateVisitorJWT(tokenString string, key []byte) (*jwt.Token, error) {
	token, err := validateJWT(tokenString, new(VisitorClaims), key)
	if err != nil {
		return nil, fmt.Errorf("validating visitor jwt token: %w", err)
	}

	_, ok := token.Claims.(*VisitorClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return token, nil
}

func generateJWT(claims jwt.Claims, key []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("generating jwt token: %w", err)
	}

	return tokenString, nil
}

func validateJWT(tokenString string, claims jwt.Claims, key []byte) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (any, error) {
		return key, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("parsing jwt token `%s`: %w", tokenString, err)
	}

	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, errors.New("unexpected signing method")
	}

	return token, nil
}
