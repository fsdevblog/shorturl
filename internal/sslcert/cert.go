package sslcert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"
)

// Generator представляет собой генератор SSL сертификатов.
// Содержит базовый шаблон сертификата.
type Generator struct {
	cert *x509.Certificate
}

// New создает новый экземпляр генератора сертификатов с настройками по умолчанию:
// - Организация: "Test Org"
// - Страна: "RU"
// - IP адреса: localhost (127.0.0.1 и ::1)
// - Срок действия: 10 лет
// - Назначение: клиентская и серверная аутентификация.
func New() (*Generator, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128)) //nolint:mnd
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}
	cert := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			Country:      []string{"RU"},
		},
		IPAddresses: []net.IP{
			net.IPv4(127, 0, 0, 1), //nolint:mnd
			net.IPv6loopback,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(10, 0, 0), //nolint:mnd
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	return &Generator{cert: cert}, nil
}

// MustNew аналогичен New(), но в случае ошибки вызывает панику.
func MustNew() *Generator {
	g, err := New()
	if err != nil {
		panic(err)
	}
	return g
}

// Modifier модификатор для изменения параметров сертификата.
type Modifier struct {
	apply func(*x509.Certificate)
}

// Modify создает новый модификатор сертификата.
// Позволяет изменять параметры сертификата перед его генерацией.
func Modify(fn func(*x509.Certificate)) Modifier {
	return Modifier{apply: fn}
}

// Generate генерирует новую пару сертификат/приватный ключ.
// Возвращает:
// - Сертификат в формате PEM
// - Приватный ключ в формате PEM
// - Ошибку, если она возникла.
func (c *Generator) Generate(modifiers ...Modifier) ([]byte, []byte, error) {
	cert := c.cert

	for _, m := range modifiers {
		m.apply(cert)
	}

	privKey, errGenPrivKey := rsa.GenerateKey(rand.Reader, 4096) //nolint:mnd
	if errGenPrivKey != nil {
		return nil, nil, fmt.Errorf("generate private key: %w", errGenPrivKey)
	}
	certBytes, errGenCert := x509.CreateCertificate(rand.Reader, cert, cert, &privKey.PublicKey, privKey)
	if errGenCert != nil {
		return nil, nil, fmt.Errorf("generate certificate: %w", errGenCert)
	}

	certPEM, privPEM, errPEM := c.pemEncode(privKey, certBytes)
	if errPEM != nil {
		return nil, nil, fmt.Errorf("encode certificate and private key: %w", errPEM)
	}
	return certPEM, privPEM, nil
}

// CheckPemFiles проверяет валидность PEM-файлов сертификата и приватного ключа.
// Проверяет:
// - Наличие данных в файлах
// - Корректность PEM-формата
// - Тип сертификата
// - Срок действия сертификата
//
// Возможные ошибки:
// - ErrBlankPEM - пустые данные в PEM-файле
// - ErrCertExpired - срок действия сертификата истек
// - ErrCertNotValidYet - сертификат еще не вступил в силу.
func (c *Generator) CheckPemFiles(certSource io.Reader, keySource io.Reader) error {
	certBytes, errReadCert := io.ReadAll(certSource)
	if errReadCert != nil {
		return fmt.Errorf("read certificate: %w", errReadCert)
	}
	if len(certBytes) == 0 {
		return ErrBlankPEM
	}

	keyBytes, errReadKey := io.ReadAll(keySource)
	if errReadKey != nil {
		return fmt.Errorf("read private key: %w", errReadKey)
	}
	if len(keyBytes) == 0 {
		return ErrBlankPEM
	}

	certPemDecoded, errCertPemDecode := c.pemDecode(certBytes)
	if errCertPemDecode != nil {
		return fmt.Errorf("decode certificate: %w", errCertPemDecode)
	}
	if certPemDecoded.Type != "CERTIFICATE" {
		return errors.New("certificate type is not CERTIFICATE")
	}

	cert, errParseCert := x509.ParseCertificate(certPemDecoded.Bytes)

	if errParseCert != nil {
		return fmt.Errorf("parse certificate: %w", errParseCert)
	}

	if cert.NotBefore.After(time.Now()) {
		return ErrCertNotValidYet
	}
	if cert.NotAfter.Before(time.Now()) {
		return ErrCertExpired
	}
	return nil
}

// pemDecode декодирует PEM-данные.
func (c *Generator) pemDecode(data []byte) (*pem.Block, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("pem decode: block is nil")
	}
	return block, nil
}

// pemEncode кодирует сертификат и приватный ключ в формат PEM.
func (c *Generator) pemEncode(privKey *rsa.PrivateKey, certBytes []byte) ([]byte, []byte, error) {
	var certPEM bytes.Buffer
	if errPemEncode := pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); errPemEncode != nil {
		return nil, nil, fmt.Errorf("pem encode certificate: %w", errPemEncode)
	}

	var privKeyPEM bytes.Buffer
	if errPemEncode := pem.Encode(&privKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	}); errPemEncode != nil {
		return nil, nil, fmt.Errorf("pem encode RSA: %w", errPemEncode)
	}

	return certPEM.Bytes(), privKeyPEM.Bytes(), nil
}
