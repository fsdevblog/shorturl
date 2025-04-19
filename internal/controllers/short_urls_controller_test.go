package controllers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fsdevblog/shorturl/internal/services"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/services/smocks"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type ContentType string

const (
	JSONContentType           ContentType = "json"
	FormURLEncodedContentType ContentType = "urlencoded"
)

type ShortURLControllerSuite struct {
	suite.Suite
	urlServMock *smocks.URLMock
	router      *gin.Engine
	config      *config.Config
}

func (s *ShortURLControllerSuite) SetupTest() {
	s.urlServMock = new(smocks.URLMock)
	appConf := config.Config{
		ServerAddress: ":80",
		BaseURL:       &url.URL{Scheme: "http", Host: "test.com:8080"},
		Logger:        logrus.New(),
	}
	s.config = &appConf
	s.router = SetupRouter(s.urlServMock, &appConf)
}

func (s *ShortURLControllerSuite) TestShortURLController_CreateShortURL() {
	validURL := "https://test.com/valid"
	invalidURL := "https://test .com/valid"
	shortIdentifier := "12345678"

	s.urlServMock.On("Create", validURL).
		Return(&models.URL{ShortIdentifier: shortIdentifier, URL: validURL}, nil)

	tests := []struct {
		name       string
		redirectTo string
		wantStatus int
	}{
		{name: "valid", redirectTo: validURL, wantStatus: http.StatusCreated},
		{name: "invalid", redirectTo: invalidURL, wantStatus: http.StatusUnprocessableEntity},
	}

	jsonFn := func(to string) io.Reader {
		jsonStr := fmt.Sprintf(`{"url": "%s"}`, to)
		return strings.NewReader(jsonStr)
	}
	bodyFn := func(to string) io.Reader {
		return strings.NewReader(to)
	}
	requests := []struct {
		rType       ContentType
		uri         string
		contentType string
		bodyFn      func(to string) io.Reader
	}{
		{rType: JSONContentType, uri: "/api/shorten", contentType: "application/json", bodyFn: jsonFn},
		{rType: FormURLEncodedContentType, uri: "/", contentType: "application/x-www-form-urlencoded", bodyFn: bodyFn},
	}
	for _, r := range requests {
		for _, tt := range tests {
			s.Run(tt.name, func() {
				res := s.makeRequest(requestFields{
					Method:      http.MethodPost,
					URL:         r.uri,
					Body:        r.bodyFn(tt.redirectTo),
					ContentType: r.contentType,
				})

				defer res.Body.Close()

				s.Equal(tt.wantStatus, res.StatusCode)

				if tt.wantStatus == http.StatusCreated {
					body, _ := io.ReadAll(res.Body)
					var shortURL string
					if r.rType == JSONContentType {
						shortURL = fmt.Sprintf(`{"result":"%s/%s"}`, s.config.BaseURL.String(), shortIdentifier)
					} else {
						shortURL = fmt.Sprintf("%s/%s", s.config.BaseURL.String(), shortIdentifier)
					}
					s.Equal(shortURL, string(body))
				}
			})
		}
	}
}

func (s *ShortURLControllerSuite) TestShortURLController_Redirect() {
	validShortID := "12345678"
	notExistShortID := "12345671"
	inValidShortID := "123"

	redirectTo := "https://test.com/test/123"

	s.urlServMock.On("GetByShortIdentifier", validShortID).
		Return(&models.URL{ShortIdentifier: validShortID, URL: redirectTo}, nil)

	s.urlServMock.On("GetByShortIdentifier", notExistShortID).
		Return(nil, services.ErrRecordNotFound)

	tests := []struct {
		name       string
		requestURI string
		wantStatus int
	}{
		{name: "valid", requestURI: validShortID, wantStatus: http.StatusTemporaryRedirect},
		{name: "invalid", requestURI: inValidShortID, wantStatus: http.StatusNotFound},
		{name: "notExistShortID", requestURI: notExistShortID, wantStatus: http.StatusNotFound},
		{name: "root page", requestURI: "", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			res := s.makeRequest(requestFields{
				Method: http.MethodGet,
				URL:    "/" + tt.requestURI,
			})

			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body)
			s.Equal(tt.wantStatus, res.StatusCode, "Answer:", string(body))
			if tt.wantStatus == http.StatusTemporaryRedirect {
				s.Equal(redirectTo, res.Header.Get("Location"))
			} else {
				s.Empty(res.Header.Get("Location"))
			}
		})
	}
	s.urlServMock.AssertNumberOfCalls(s.T(), "GetByShortIdentifier", 2)
}

func (s *ShortURLControllerSuite) Test_validateURL() {
	validRaw := "https://test.com"
	validLocalhostRaw := "https://localhost"
	validIPRaw := "https://123.123.123.123/test"

	valid, _ := url.Parse(validRaw)
	validLocalhost, _ := url.Parse(validLocalhostRaw)
	validIP, _ := url.Parse(validIPRaw)

	tests := []struct {
		name    string
		rawURL  string
		want    *url.URL
		wantErr bool
	}{
		{name: "valid url", rawURL: validRaw, want: valid, wantErr: false},
		{name: "wrong scheme", rawURL: "test://test.com", want: nil, wantErr: true},
		{name: "space into", rawURL: "https://tes t.com", want: nil, wantErr: true},
		{name: "wrong chars", rawURL: "https://tes😀t.com", want: nil, wantErr: true},
		{name: "empty zone", rawURL: "https://test.", want: nil, wantErr: true},
		{name: "empty zone", rawURL: "https://test", want: nil, wantErr: true},
		{name: "localhost", rawURL: validLocalhostRaw, want: validLocalhost, wantErr: false},
		{name: "ip address", rawURL: validIPRaw, want: validIP, wantErr: false},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := validateURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				s.Failf("validateURL() `%s` error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			s.Equal(tt.want, got)
		})
	}
}

type requestFields struct {
	Method      string
	URL         string
	Body        io.Reader
	ContentType string
}

// makeRequest вспомогательная функция создающая тестовый http запрос.
func (s *ShortURLControllerSuite) makeRequest(fields requestFields) *http.Response {
	request := httptest.NewRequest(fields.Method, fields.URL, fields.Body)
	if fields.ContentType != "" {
		request.Header.Set("Content-Type", fields.ContentType)
	}
	recorder := httptest.NewRecorder()

	s.router.ServeHTTP(recorder, request)

	return recorder.Result()
}

func TestShortURLControllerSuite(t *testing.T) {
	suite.Run(t, new(ShortURLControllerSuite))
}
