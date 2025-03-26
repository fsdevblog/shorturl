package controllers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/fsdevblog/shorturl/internal/app/models"
	"github.com/fsdevblog/shorturl/internal/app/services/smocks"
	"github.com/stretchr/testify/assert"
)

func TestShortURLController_createShortURL(t *testing.T) {
	urlServMock := new(smocks.URLMock)

	validURL := "https://test.com/valid"
	invalidURL := "https://test .com/valid"
	shortIdentifier := "12345678"
	serverHostname := "example.com"

	urlServMock.On("Create", validURL).
		Return(&models.URL{ShortIdentifier: shortIdentifier, URL: validURL}, nil)

	tests := []struct {
		name       string
		shortURL   string
		wantStatus int
	}{
		{name: "valid", shortURL: validURL, wantStatus: http.StatusCreated},
		{name: "invalid", shortURL: invalidURL, wantStatus: http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := makeRequest(mRequestFields{
				method: http.MethodPost,
				url:    "/",
				body:   strings.NewReader(tt.shortURL),
				f: func(w *httptest.ResponseRecorder, r *http.Request) {
					NewShortURLController(urlServMock).createShortURL(w, r)
				},
			})

			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)

			if tt.wantStatus == http.StatusCreated {
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, fmt.Sprintf("http://%s/%s", serverHostname, shortIdentifier), string(body))
			}
		})
	}
}

func TestShortURLController_redirect(t *testing.T) {
	urlServMock := new(smocks.URLMock)

	validShortID := "12345678"
	notExistShortID := "12345671"
	inValidShortID := "123"

	redirectTo := "https://test.com/test/123"

	urlServMock.On("GetByShortIdentifier", validShortID).
		Return(&models.URL{ShortIdentifier: validShortID, URL: redirectTo}, nil)

	urlServMock.On("GetByShortIdentifier", notExistShortID).
		Return(nil, gorm.ErrRecordNotFound)

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
		t.Run(tt.name, func(t *testing.T) {
			res := makeRequest(mRequestFields{
				method: http.MethodGet,
				url:    "/" + tt.requestURI,
				body:   nil,
				f: func(w *httptest.ResponseRecorder, r *http.Request) {
					NewShortURLController(urlServMock).redirect(w, r)
				},
			})
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantStatus == http.StatusTemporaryRedirect {
				assert.Equal(t, redirectTo, res.Header.Get("Location"))
			} else {
				assert.Empty(t, res.Header.Get("Location"))
			}
		})
	}
	urlServMock.AssertNumberOfCalls(t, "GetByShortIdentifier", 2)
}

func Test_validateURL(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL() `%s` error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !assert.Equal(t, tt.want, got) {
				t.Errorf("validateURL() `%s` got = %v, wantErr %v", tt.name, got, tt.want)
			}
		})
	}
}

type mRequestFields struct {
	method string
	url    string
	body   io.Reader
	f      func(*httptest.ResponseRecorder, *http.Request)
}

func makeRequest(fields mRequestFields) *http.Response {
	request := httptest.NewRequest(fields.method, fields.url, fields.body)
	recorder := httptest.NewRecorder()

	fields.f(recorder, request)

	return recorder.Result()
}
