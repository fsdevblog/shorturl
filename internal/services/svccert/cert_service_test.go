package svccert

import (
	"os"
	"path"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/suite"
)

type CertServiceSuite struct {
	suite.Suite
	cert        *Cert
	certTmpFile *os.File
	keyTmpFile  *os.File
}

func TestCertService(t *testing.T) {
	suite.Run(t, new(CertServiceSuite))
}

func (s *CertServiceSuite) SetupTest() {
	var errCertTmp, errKeyTmp error

	s.certTmpFile, errCertTmp = os.CreateTemp("", "cert.pem")
	s.Require().NoError(errCertTmp)

	s.keyTmpFile, errKeyTmp = os.CreateTemp("", "key.pem")
	s.Require().NoError(errKeyTmp)

	s.cert = New(func(opt *Options) {
		opt.CertFilePath = s.certTmpFile.Name()
		opt.KeyFilePath = s.keyTmpFile.Name()
	})
}

func (s *CertServiceSuite) TearDownTest() {
	errCertTmpClose := s.certTmpFile.Close()
	s.Require().NoError(errCertTmpClose)

	errKeyTmpClose := s.keyTmpFile.Close()
	s.Require().NoError(errKeyTmpClose)

	_ = os.Remove(s.certTmpFile.Name())
	_ = os.Remove(s.keyTmpFile.Name())
}

func (s *CertServiceSuite) TestGenerateAndSaveIfNeed() {
	s.Run("blank pem files", func() {
		err := s.cert.GenerateAndSaveIfNeed()
		s.Require().NoError(err)

		// проверяем, данные должны записаться во временный файл.
		certBytes, errReadCertFile := os.ReadFile(s.certTmpFile.Name())
		s.Require().NoError(errReadCertFile)
		s.Require().NotEmpty(certBytes)

		keyBytes, errReadKeyFile := os.ReadFile(s.keyTmpFile.Name())
		s.Require().NoError(errReadKeyFile)
		s.Require().NotEmpty(keyBytes)
	})

	s.TearDownTest()
	s.SetupTest()

	s.Run("valid pem", func() {
		certPEM, errReadCertFile := os.ReadFile("testdata/valid_cert.pem")
		s.Require().NoError(errReadCertFile)

		keyPEM, errReadKeyFile := os.ReadFile("testdata/valid_key.pem")
		s.Require().NoError(errReadKeyFile)

		// копируем данные во временные файлы.
		s.Require().NoError(os.WriteFile(s.certTmpFile.Name(), certPEM, 0600))
		s.Require().NoError(os.WriteFile(s.keyTmpFile.Name(), keyPEM, 0600))

		// тестируем метод.
		errGen := s.cert.GenerateAndSaveIfNeed()
		s.Require().NoError(errGen)

		certResult, errReadCertResult := os.ReadFile(s.certTmpFile.Name())
		s.Require().NoError(errReadCertResult)

		keyResult, errReadKeyResult := os.ReadFile(s.keyTmpFile.Name())
		s.Require().NoError(errReadKeyResult)

		// Новый сертификат не должен сгенерироваться, а значит данные должны остаться прежними
		s.Equal(certResult, certPEM)
		s.Equal(keyResult, keyPEM)
	})
}

func (s *CertServiceSuite) TestOpenFiles() {
	s.Run("valid and unexisted path", func() {
		certPath := path.Join(os.TempDir(), gofakeit.Word(), "cert.pem")
		keyPath := path.Join(os.TempDir(), gofakeit.Word(), "key.pem")

		certFile, keyFile, err := openFiles(certPath, keyPath)
		s.Require().NoError(err)
		_ = certFile.Close()
		_ = keyFile.Close()

		// файлы должны создаться.
		_, errCertStat := os.Stat(certPath)
		s.Require().NoError(errCertStat)

		_, errKeyStat := os.Stat(keyPath)
		s.Require().NoError(errKeyStat)
	})
}
