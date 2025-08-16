package svccert

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fsdevblog/shorturl/internal/sslcert"
)

const (
	defaultCertFilePath = "cert.pem" // Путь к файлу сертификата по умолчанию.
	defaultKeyFilePath  = "key.pem"  // Путь к файлу приватного ключа по умолчанию.
)

// Options Опции для конфигурации инициализатора.
type Options struct {
	CertFilePath string // Путь к файлу сертификата.
	KeyFilePath  string // Путь к файлу приватного ключа.
}

// Cert Сервис для генерации SSL/TLS сертификатов.
type Cert struct {
	gen          *sslcert.Generator
	certFilePath string
	keyFilePath  string
}

// New создает новый экземпляр Cert с указанными опциями.
// Если опции не указаны, используются значения по умолчанию.
//
// Параметры:
//   - opts: вариативный список функций для настройки опций.
//
// Возвращает:
//   - *Cert: новый экземпляр структуры Cert.
func New(opts ...func(*Options)) *Cert {
	defaultOpts := Options{
		CertFilePath: defaultCertFilePath,
		KeyFilePath:  defaultKeyFilePath,
	}
	for _, opt := range opts {
		opt(&defaultOpts)
	}
	return &Cert{
		gen:          sslcert.MustNew(),
		certFilePath: defaultOpts.CertFilePath,
		keyFilePath:  defaultOpts.KeyFilePath,
	}
}

// CertString возвращает содержимое файла сертификата в виде строки.
//
// Возвращает:
//   - string: содержимое сертификата.
//   - error: ошибка при чтении файла.
func (c *Cert) CertString() (string, error) {
	cert, err := os.ReadFile(c.certFilePath)
	if err != nil {
		return "", fmt.Errorf("read certificate file: %w", err)
	}
	return string(cert), nil
}

// KeyString возвращает содержимое файла приватного ключа в виде строки.
//
// Возвращает:
//   - string: содержимое приватного ключа.
//   - error: ошибка при чтении файла.
func (c *Cert) KeyString() (string, error) {
	key, err := os.ReadFile(c.keyFilePath)
	if err != nil {
		return "", fmt.Errorf("read key file: %w", err)
	}
	return string(key), nil
}

// PairString возвращает пару сертификат/ключ в виде строк.
//
// Возвращает:
//   - string: содержимое сертификата.
//   - string: содержимое приватного ключа.
//   - error: ошибка при чтении файлов.
func (c *Cert) PairString() (string, string, error) {
	key, errKey := c.KeyString()
	if errKey != nil {
		return "", "", errKey
	}
	cert, errCert := c.CertString()
	if errCert != nil {
		return "", "", errCert
	}
	return cert, key, nil
}

// GenerateAndSaveIfNeed проверяет существующие файлы сертификата и ключа.
// Если файлы отсутствуют, пусты или сертификат просрочен - генерирует новую пару.
//
// Параметры:
//   - modifiers: модификаторы для настройки генерации сертификата.
//
// Возвращает:
//   - error: ошибка при проверке/генерации/сохранении сертификата.
func (c *Cert) GenerateAndSaveIfNeed(modifiers ...sslcert.Modifier) error {
	cert, key, errPaths := openFiles(c.certFilePath, c.keyFilePath)
	if errPaths != nil {
		return fmt.Errorf("open certificate and private key files: %w", errPaths)
	}
	defer cert.Close()
	defer key.Close()

	errCheck := c.gen.CheckPemFiles(cert, key)
	if errCheck != nil {
		if errors.Is(errCheck, sslcert.ErrBlankPEM) || errors.Is(errCheck, sslcert.ErrCertExpired) {
			return c.generateAndSave(cert, key, modifiers...)
		}
		return fmt.Errorf("check certificate and private key: %w", errCheck)
	}
	return nil
}

// generateAndSave создает новую пару сертификат/ключ и сохраняет их.
func (c *Cert) generateAndSave(cert io.Writer, key io.Writer, modifiers ...sslcert.Modifier) error {
	certPEM, keyPEM, errGen := c.gen.Generate(modifiers...)
	if errGen != nil {
		return fmt.Errorf("generate certificate and private key: %w", errGen)
	}
	errSave := c.savePemToFiles(saveParams{
		CertIO:    cert,
		KeyIO:     key,
		CertBytes: certPEM,
		PrivKey:   keyPEM,
	})
	if errSave != nil {
		return fmt.Errorf("save certificate and private key: %w", errSave)
	}
	return nil
}

// saveParams содержит параметры для сохранения сертификата и ключа.
type saveParams struct {
	CertIO    io.Writer // Writer для записи сертификата.
	KeyIO     io.Writer // Writer для записи ключа.
	CertBytes []byte    // Байты сертификата.
	PrivKey   []byte    // Байты приватного ключа.
}

// savePemToFiles сохраняет сертификат и ключ в файлы.
func (c *Cert) savePemToFiles(info saveParams) error {
	_, errCertWrite := info.CertIO.Write(info.CertBytes)
	if errCertWrite != nil {
		return fmt.Errorf("save certificate: %w", errCertWrite)
	}
	_, errKeyWrite := info.KeyIO.Write(info.PrivKey)
	if errKeyWrite != nil {
		return fmt.Errorf("save private key: %w", errKeyWrite)
	}
	return nil
}

// openFiles открывает файлы сертификата и ключа по указанным путям.
// Если файлы не существуют - создает их.
//
// Параметры:
//   - certPath: путь к файлу сертификата.
//   - keyPath: путь к файлу ключа.
//
// Возвращает:
//   - *os.File: файл сертификата.
//   - *os.File: файл ключа.
//   - error: ошибка при открытии/создании файлов.
func openFiles(certPath string, keyPath string) (*os.File, *os.File, error) {
	// Создаем директории для сертификата и ключа
	certDir := filepath.Dir(certPath)
	if errCertDir := os.MkdirAll(certDir, 0755); errCertDir != nil {
		return nil, nil, fmt.Errorf("create certificate directory: %w", errCertDir)
	}

	keyDir := filepath.Dir(keyPath)
	if errKeyDir := os.MkdirAll(keyDir, 0755); errKeyDir != nil {
		return nil, nil, fmt.Errorf("create key directory: %w", errKeyDir)
	}

	_, errCert := os.Stat(certPath)
	if errCert != nil && !errors.Is(errCert, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("stat certificate file: %w", errCert)
	}

	_, errKey := os.Stat(keyPath)
	if errKey != nil && !errors.Is(errKey, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("stat key file: %w", errKey)
	}

	certFile, errCertOpen := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE, 0600)
	if errCertOpen != nil {
		return nil, nil, fmt.Errorf("open certificate file: %w", errCertOpen)
	}

	keyFile, errKeyOpen := os.OpenFile(keyPath, os.O_RDWR|os.O_CREATE, 0600)
	if errKeyOpen != nil {
		certFile.Close()
		return nil, nil, fmt.Errorf("open key file: %w", errKeyOpen)
	}

	return certFile, keyFile, nil
}
