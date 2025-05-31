package controllers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/fsdevblog/shorturl/internal/tokens"

	"github.com/fsdevblog/shorturl/internal/controllers/mocksctrl"
	"github.com/golang/mock/gomock"

	"github.com/fsdevblog/shorturl/internal/logs"

	"github.com/fsdevblog/shorturl/internal/services"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type CType string

const (
	JSONCType   CType = "json"
	URLEncCType CType = "urlencoded"
	jwtSecret         = "test secret"
)

type ShortURLControllerSuite struct {
	suite.Suite
	mockShortURLStore *mocksctrl.MockShortURLStore
	router            *gin.Engine
	config            *config.Config
}

func (s *ShortURLControllerSuite) SetupTest() {
	mockShortURL := gomock.NewController(s.T())
	defer mockShortURL.Finish()

	if seedErr := gofakeit.Seed(0); seedErr != nil {
		s.T().Fatal(seedErr)
	}

	s.mockShortURLStore = mocksctrl.NewMockShortURLStore(mockShortURL)

	appConf := config.Config{
		ServerAddress:    ":80",
		BaseURL:          &url.URL{Scheme: "http", Host: "test.com:8080"},
		VisitorJWTSecret: jwtSecret,
	}
	s.config = &appConf
	s.router = SetupRouter(RouterParams{
		URLService:  s.mockShortURLStore,
		PingService: nil,
		AppConf:     appConf,
		Logger:      logs.New(os.Stdout),
	})
}

func (s *ShortURLControllerSuite) TestShortURLController_CreateBatch() {
	if seedErr := gofakeit.Seed(0); seedErr != nil {
		s.T().Fatal(seedErr)
	}

	uniqData := s.prepareTestForCreateBatch(3, true)
	notUniqData := s.prepareTestForCreateBatch(3, false)

	tests := []struct {
		name           string
		wantStatus     int
		requestPayload []BatchCreateParams
		mockResponse   *services.BatchCreateShortURLsResponse
		apiResponse    []BatchCreateResponse
	}{
		{name: "uniq urls", wantStatus: http.StatusCreated, requestPayload: uniqData.requestPayload,
			mockResponse: uniqData.mockResponse, apiResponse: uniqData.apiExpectResponse},
		{name: "not uniq", wantStatus: http.StatusConflict, requestPayload: notUniqData.requestPayload,
			mockResponse: notUniqData.mockResponse, apiResponse: notUniqData.apiExpectResponse},
	}

	for _, t := range tests {
		s.Run(t.name, func() {
			s.mockShortURLStore.EXPECT().
				BatchCreate(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(t.mockResponse, nil).
				Times(1)

			payload, _ := json.Marshal(t.requestPayload)
			res := s.makeRequest(requestFields{
				Method: http.MethodPost,
				URL:    "/api/shorten/batch",
				Body:   bytes.NewReader(payload),
			},
				withContentType("application/json"),
				withGzip(true),
			)

			defer func() {
				closeErr := res.Body.Close()
				s.Require().NoError(closeErr)
			}()

			body, readBodyErr := readBody(res.Body, true)
			s.Require().NoError(readBodyErr)

			s.Equal(t.wantStatus, res.StatusCode)

			var respBody []BatchCreateResponse
			bodyJSONErr := json.Unmarshal(body, &respBody)

			s.Require().NoError(bodyJSONErr)

			s.Equal(t.apiResponse, respBody)
		})
	}
}

type prepareTestDataForCreateBatch struct {
	mockResponse      *services.BatchCreateShortURLsResponse
	apiExpectResponse []BatchCreateResponse
	requestPayload    []BatchCreateParams
}

func (s *ShortURLControllerSuite) prepareTestForCreateBatch(batchSize int, isUniq bool) *prepareTestDataForCreateBatch {
	var urls = make([]string, 0, batchSize)
	for range batchSize {
		urls = append(urls, gofakeit.URL())
	}

	var reqData = make([]BatchCreateParams, batchSize)
	batchResponse := services.NewBatchExecResponseURL(
		services.NewBatchExecResponse[models.URL](batchSize),
	)
	var expectedResponse = make([]BatchCreateResponse, batchSize)

	for i, rawURL := range urls {
		reqData[i] = BatchCreateParams{
			CorrelationID: gofakeit.UUID(),
			OriginalURL:   rawURL,
		}
		randSid, _ := gofakeit.Generate("????????")
		item := models.URL{
			URL:             rawURL,
			ShortIdentifier: randSid,
		}

		var rErr error
		if !isUniq && i == len(urls)-1 {
			// Симулируем поведение при неуникальной ссылке на последней итерации.
			rErr = services.ErrDuplicateKey
		}
		batchResponse.Set(services.BatchResponseItem[models.URL]{
			Item: item,
			Err:  rErr,
		}, i)

		expectedResponse[i] = BatchCreateResponse{
			CorrelationID: reqData[i].CorrelationID,
			ShortURL:      s.genShortURLForSid(randSid),
		}
	}

	return &prepareTestDataForCreateBatch{
		mockResponse:      batchResponse,
		apiExpectResponse: expectedResponse,
		requestPayload:    reqData,
	}
}

func (s *ShortURLControllerSuite) TestShortURLController_UserURLs() {
	visitorWithURLs := gofakeit.UUID()
	jwtTokenWithURLs, jwtTokenWithURLsErr := tokens.GenerateVisitorJWT(visitorWithURLs, time.Hour,
		[]byte(s.config.VisitorJWTSecret))

	s.Require().NoError(jwtTokenWithURLsErr)

	visitorWithoutURLs := gofakeit.UUID()
	jwtTokenWithoutURLs, jwtTokenWithoutURLsErr := tokens.GenerateVisitorJWT(visitorWithoutURLs, time.Hour,
		[]byte(s.config.VisitorJWTSecret))

	s.Require().NoError(jwtTokenWithoutURLsErr)

	s.mockShortURLStore.EXPECT().GetAllByVisitorUUID(gomock.Any(), visitorWithURLs).
		Return([]models.URL{
			{
				ShortIdentifier: "12345678",
				URL:             "https://test.com/test/123",
			},
			{
				ShortIdentifier: "12345679",
				URL:             "https://test.com/test/124",
			},
		}, nil)

	s.mockShortURLStore.
		EXPECT().
		GetAllByVisitorUUID(gomock.Any(), visitorWithoutURLs).Return([]models.URL{}, nil)

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "with_urls", wantStatus: http.StatusOK, token: jwtTokenWithURLs},
		{name: "without_urls", wantStatus: http.StatusNoContent, token: jwtTokenWithoutURLs},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			res := s.makeRequest(requestFields{
				Method: http.MethodGet,
				URL:    "/api/user/urls",
			},
				withCookies([]*http.Cookie{
					{
						Name:  "visitor",
						Value: tt.token,
					},
				}),
			)
			defer func() {
				if err := res.Body.Close(); err != nil {
					s.T().Fatal(err)
				}
			}()
			s.Equalf(tt.wantStatus, res.StatusCode,
				"%s wrong status code, want %d, got %d", tt.name, tt.wantStatus, res.StatusCode)
		})
	}
}

//nolint:gocognit
func (s *ShortURLControllerSuite) TestShortURLController_CreateShortURL() {
	validURL := "https://test.com/valid"
	notUniqURL := "https://test.com/not_uniq"
	invalidURL := "https://test .com/valid"
	shortIdentifier := "12345678"

	s.mockShortURLStore.EXPECT().
		Create(gomock.Any(), gomock.Any(), validURL).
		Return(&models.URL{
			URL:             validURL,
			ShortIdentifier: shortIdentifier,
		}, true, nil).MinTimes(1)

	s.mockShortURLStore.EXPECT().
		Create(gomock.Any(), gomock.Any(), notUniqURL).
		Return(&models.URL{
			URL:             notUniqURL,
			ShortIdentifier: shortIdentifier,
		}, false, nil).
		MinTimes(1)

	tests := []struct {
		name       string
		redirectTo string
		wantStatus int
	}{
		{name: "valid", redirectTo: validURL, wantStatus: http.StatusCreated},
		{name: "invalid", redirectTo: invalidURL, wantStatus: http.StatusUnprocessableEntity},
		{name: "not_uniq", redirectTo: notUniqURL, wantStatus: http.StatusConflict},
	}

	jsonFn := func(to string) io.Reader {
		jsonStr := fmt.Sprintf(`{"url": "%s"}`, to)
		return strings.NewReader(jsonStr)
	}
	bodyFn := func(to string) io.Reader {
		return strings.NewReader(to)
	}
	requests := []struct {
		rType       CType
		uri         string
		contentType string
		bodyFn      func(to string) io.Reader
		gzip        bool
	}{
		{rType: JSONCType, uri: "/api/shorten", contentType: "application/json", bodyFn: jsonFn, gzip: true},
		{rType: JSONCType, uri: "/api/shorten", contentType: "application/json", bodyFn: jsonFn, gzip: false},
		{rType: URLEncCType, uri: "/", contentType: "application/x-www-form-urlencoded", bodyFn: bodyFn, gzip: true},
		{rType: URLEncCType, uri: "/", contentType: "application/x-www-form-urlencoded", bodyFn: bodyFn, gzip: false},
	}
	for _, r := range requests {
		for _, tt := range tests {
			s.Run(tt.name, func() {
				res := s.makeRequest(requestFields{
					Method: http.MethodPost,
					URL:    r.uri,
					Body:   r.bodyFn(tt.redirectTo),
				},
					withContentType(r.contentType),
					withGzip(r.gzip),
				)

				defer func() {
					if err := res.Body.Close(); err != nil {
						s.T().Fatal(err)
					}
				}()

				s.Equal(tt.wantStatus, res.StatusCode)

				if tt.wantStatus == http.StatusCreated || tt.wantStatus == http.StatusConflict {
					body, bErr := readBody(res.Body, r.gzip)

					if bErr != nil {
						s.T().Fatalf("failed to read body: %v", bErr)
					}
					var shortURL string
					if r.rType == JSONCType {
						shortURL = fmt.Sprintf(`{"result":"%s/%s"}`, s.config.BaseURL.String(), shortIdentifier)
					} else {
						shortURL = s.genShortURLForSid(shortIdentifier)
					}
					s.Equal(shortURL, string(body))
				}

				if r.gzip {
					s.Equal("gzip", res.Header.Get("Content-Encoding"))
				}
			})
		}
	}
}

func (s *ShortURLControllerSuite) TestShortURLController_Redirect() {
	validShortID := "12345678"
	notExistShortID := "12345671"
	inValidShortID := "123"
	deletedSID := "deleted1"

	redirectTo := "https://test.com/test/123"

	s.mockShortURLStore.EXPECT().
		GetByShortIdentifier(gomock.Any(), validShortID).
		Return(&models.URL{ShortIdentifier: validShortID, URL: redirectTo}, nil).
		Times(1)

	s.mockShortURLStore.EXPECT().
		GetByShortIdentifier(gomock.Any(), notExistShortID).
		Return(nil, services.ErrRecordNotFound).
		Times(1)
	now := time.Now()
	s.mockShortURLStore.EXPECT().
		GetByShortIdentifier(gomock.Any(), deletedSID).
		Return(&models.URL{
			DeletedAt:       &now,
			URL:             gofakeit.URL(),
			ShortIdentifier: deletedSID,
		}, nil)

	tests := []struct {
		name       string
		requestURI string
		wantStatus int
	}{
		{name: "valid", requestURI: validShortID, wantStatus: http.StatusTemporaryRedirect},
		{name: "invalid", requestURI: inValidShortID, wantStatus: http.StatusNotFound},
		{name: "notExistShortID", requestURI: notExistShortID, wantStatus: http.StatusNotFound},
		{name: "root page", requestURI: "", wantStatus: http.StatusNotFound},
		{name: "deleted", requestURI: deletedSID, wantStatus: http.StatusGone},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			res := s.makeRequest(requestFields{
				Method: http.MethodGet,
				URL:    "/" + tt.requestURI,
			})

			defer func() {
				if err := res.Body.Close(); err != nil {
					s.T().Fatal(err)
				}
			}()

			body, _ := io.ReadAll(res.Body)
			s.Equal(tt.wantStatus, res.StatusCode, "Answer:", string(body))
			if tt.wantStatus == http.StatusTemporaryRedirect {
				s.Equal(redirectTo, res.Header.Get("Location"))
			} else {
				s.Empty(res.Header.Get("Location"))
			}
		})
	}
}

func (s *ShortURLControllerSuite) TestShortURLController_DeleteUserURLs() {
	size := 100
	visitorUUID := gofakeit.UUID()
	jwtTokenString, jwtTokenErr := tokens.GenerateVisitorJWT(visitorUUID, time.Hour,
		[]byte(s.config.VisitorJWTSecret))

	s.Require().NoError(jwtTokenErr)
	var validShortIDs = make([]string, 0, size)
	for range size {
		gen, genErr := gofakeit.Generate("????????")
		if genErr != nil {
			s.Require().NoError(genErr)
		}
		validShortIDs = append(validShortIDs, gen)
	}
	withUnexistShortID := append(validShortIDs, "un_exist") //nolint:gocritic

	// в обоих случаях должен вернуться статус 202, независимо от того существуют записи или нет.
	var tests = []struct {
		name       string
		wantStatus int
		shortIDs   []string
	}{
		{
			name:       "exist ids",
			wantStatus: http.StatusAccepted,
			shortIDs:   validShortIDs,
		}, {
			name:       "not exist ids",
			wantStatus: http.StatusAccepted,
			shortIDs:   withUnexistShortID,
		},
	}

	s.mockShortURLStore.EXPECT().MarkAsDeleted(gomock.Any(), validShortIDs, visitorUUID).Return(nil)
	s.mockShortURLStore.EXPECT().MarkAsDeleted(gomock.Any(), withUnexistShortID, visitorUUID).Return(nil)

	for _, test := range tests {
		s.Run(test.name, func() {
			res := s.makeRequest(requestFields{
				Method: http.MethodDelete,
				URL:    "/api/user/urls",
				Body: strings.NewReader(
					fmt.Sprintf(`["%s"]`, strings.Join(test.shortIDs, `", "`)),
				),
			},
				withContentType("application/json"),
				withCookies([]*http.Cookie{
					{
						Name:  "visitor",
						Value: jwtTokenString,
					},
				}),
			)
			defer func() {
				if err := res.Body.Close(); err != nil {
					s.T().Fatal(err)
				}
			}()
			s.Equalf(test.wantStatus, res.StatusCode,
				"%s wrong status code, want %d, got %d", test.name, test.wantStatus, res.StatusCode)
		})
	}
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
	Method string
	URL    string
	Body   io.Reader
}

type requestOptions struct {
	contentType string
	gziped      bool
	cookies     []*http.Cookie
}

func withContentType(ct string) func(*requestOptions) {
	return func(fn *requestOptions) {
		fn.contentType = ct
	}
}

func withGzip(b bool) func(*requestOptions) {
	return func(fn *requestOptions) {
		fn.gziped = b
	}
}

func withCookies(c []*http.Cookie) func(*requestOptions) {
	return func(fn *requestOptions) {
		fn.cookies = c
	}
}

// makeRequest вспомогательная функция создающая тестовый http запрос.
func (s *ShortURLControllerSuite) makeRequest(fields requestFields, opts ...func(*requestOptions)) *http.Response {
	options := requestOptions{
		contentType: "text/plain",
		gziped:      false,
		cookies:     nil,
	}
	for _, opt := range opts {
		opt(&options)
	}

	var body io.Reader
	if fields.Body != nil {
		body = fields.Body
	}

	// Добавляем gzip сжатие тела запроса, если надо.
	if options.gziped && fields.Body != nil {
		var gzipBuffer bytes.Buffer
		gzipW, gzErr := gzip.NewWriterLevel(&gzipBuffer, gzip.BestSpeed)
		if gzErr != nil {
			s.T().Fatalf("failed to create gzip writer: %v", gzErr)
		}

		// копируем тело в gzip.Writer.
		_, copyErr := io.Copy(gzipW, fields.Body)
		if copyErr != nil {
			s.T().Fatalf("failed to copy request body to gzip writer: %v", copyErr)
		}

		if err := gzipW.Close(); err != nil {
			s.T().Fatalf("failed to close gzip writer: %v", err)
		}
		body = &gzipBuffer
	}

	request := httptest.NewRequest(fields.Method, fields.URL, body)
	if options.contentType != "" {
		request.Header.Set("Content-Type", options.contentType)
	}
	if options.gziped {
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Accept-Encoding", "gzip")
	}

	if options.cookies != nil {
		for _, cookie := range options.cookies {
			request.AddCookie(cookie)
		}
	}

	recorder := httptest.NewRecorder()

	s.router.ServeHTTP(recorder, request)

	return recorder.Result()
}

func TestShortURLControllerSuite(t *testing.T) {
	suite.Run(t, new(ShortURLControllerSuite))
}

func (s *ShortURLControllerSuite) genShortURLForSid(sid string) string {
	return fmt.Sprintf("%s/%s", s.config.BaseURL.String(), sid)
}

func unGzip(r io.Reader) ([]byte, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	body, err := io.ReadAll(gzr)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// readBody Читает тело запроса, если тело сжатое - расжимает.
func readBody(r io.Reader, compressed bool) ([]byte, error) {
	var body []byte
	var bErr error
	if compressed {
		body, bErr = unGzip(r)
		if bErr != nil {
			return nil, bErr
		}
		return body, nil
	}
	body, bErr = io.ReadAll(r)
	if bErr != nil {
		return nil, bErr
	}
	return body, nil
}
