package sslcert

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CertSuite struct {
	suite.Suite
	gen *Generator
}

func TestCertSuite(t *testing.T) {
	suite.Run(t, new(CertSuite))
}

func (s *CertSuite) SetupTest() {
	s.gen = MustNew()
}

func (s *CertSuite) TestGenerate() {
	certPEM, keyPEM, err := s.gen.Generate()
	s.Require().NoError(err)
	s.Require().NotEmpty(certPEM)
	s.Require().NotEmpty(keyPEM)
}

func (s *CertSuite) TestCheckPEMFiles() {
	s.Run("blank pem", func() {
		cert := new(bytes.Buffer)
		key := new(bytes.Buffer)
		err := s.gen.CheckPemFiles(cert, key)
		s.Require().Error(err)
		s.Require().Equal(ErrBlankPEM, err)
	})

	s.Run("expired gen", func() {
		expiredCertPEM, errReadExpiredCert := os.OpenFile("testdata/expired_cert.pem", os.O_RDONLY, 0644)
		defer func() {
			if expiredCertPEM != nil {
				expiredCertPEM.Close()
			}
		}()
		s.Require().NoError(errReadExpiredCert)

		expiredKeyPEM, errReadExpiredKey := os.OpenFile("testdata/expired_key.pem", os.O_RDONLY, 0644)
		defer func() {
			if expiredKeyPEM != nil {
				expiredKeyPEM.Close()
			}
		}()
		s.Require().NoError(errReadExpiredKey)

		errCheck := s.gen.CheckPemFiles(expiredCertPEM, expiredKeyPEM)
		s.Require().ErrorIs(errCheck, ErrCertExpired)
	})
}
