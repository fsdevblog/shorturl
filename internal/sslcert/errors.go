package sslcert

import "errors"

var (
	ErrCertExpired     = errors.New("certificate is expired")       // Срок действия сертификата истек.
	ErrCertNotValidYet = errors.New("certificate is not valid yet") // Сертификат еще не вступил в силу.
	ErrBlankPEM        = errors.New("pem is blank")                 // Пустые данные в PEM-файле.
)
