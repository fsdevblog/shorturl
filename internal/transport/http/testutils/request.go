package testutils

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

// Params основные параметры запроса.
type Params struct {
	Method string
	URL    string
	Body   io.Reader
}

// Options дополнительный опции запроса.
type Options struct {
	ContentType string
	Gziped      bool
	Cookies     []*http.Cookie
	Headers     map[string]string
}

// RequestWithTypedReturn выполняет HTTP-запрос, читает его ответ и преобразует ответ в заданный тип (generic).
//
// r     - объект *gin.Engine для обработки HTTP-запроса.
// fields - основные параметры запроса (метод, URL, тело).
// opts   - функции для настройки дополнительных опций запроса.
//
// Возвращает типизированный результат (generic), объект http.Response и ошибку, если произошла ошибка.
//
// Пример:
//
//	type ResponseData struct {
//	    Key string `json:"key"`
//	}
//	result, resp, err := RequestWithTypedReturn[ResponseData](engine, Params{
//	    Method: "GET",
//	    URL:    "/test",
//	})
func RequestWithTypedReturn[T any](
	r *gin.Engine,
	fields Params,
	opts ...func(*Options),
) (T, *http.Response, error) {
	var typedResult T
	response, err := Request(r, fields, opts...)
	if err != nil {
		return typedResult, nil, fmt.Errorf("failed to request: %w", err)
	}
	defer response.Body.Close()

	data, errRead := io.ReadAll(response.Body)
	if errRead != nil {
		return typedResult, nil, fmt.Errorf("read response body error: %w", errRead)
	}
	errUnmarshal := json.Unmarshal(data, &typedResult)
	if errUnmarshal != nil {
		return typedResult, nil, fmt.Errorf("unmarshal response body error: %w", errUnmarshal)
	}
	return typedResult, response, nil
}

// Request выполняет HTTP-запрос с заданными параметрами и опциями.
//
// r - объект *gin.Engine для обработки запроса.
// fields - основные параметры запроса (метод, URL, тело).
// opts - функции для настройки дополнительных опций запроса.
//
// Возвращает объект http.Response и ошибку, если произошла ошибка.
func Request(r *gin.Engine, fields Params, opts ...func(*Options)) (*http.Response, error) {
	options := Options{
		ContentType: "text/plain",
		Gziped:      false,
	}
	for _, opt := range opts {
		opt(&options)
	}

	var body io.Reader
	if fields.Body != nil {
		body = fields.Body
	}

	// Добавляем gzip сжатие тела запроса, если надо.
	if options.Gziped && fields.Body != nil {
		var gzipBuffer bytes.Buffer
		gzipW, gzErr := gzip.NewWriterLevel(&gzipBuffer, gzip.BestSpeed)
		if gzErr != nil {
			return nil, fmt.Errorf("failed to create gzip writer: %w", gzErr)
		}

		// копируем тело в gzip.Writer.
		_, copyErr := io.Copy(gzipW, fields.Body)
		if copyErr != nil {
			return nil, fmt.Errorf("failed to copy request body to gzip writer: %w", copyErr)
		}

		if err := gzipW.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}
		body = &gzipBuffer
	}

	request := httptest.NewRequest(fields.Method, fields.URL, body)
	if options.ContentType != "" {
		request.Header.Set("Content-Type", options.ContentType)
	}
	if options.Gziped {
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Accept-Encoding", "gzip")
	}

	for _, cookie := range options.Cookies {
		request.AddCookie(cookie)
	}

	for n, v := range options.Headers {
		request.Header.Set(n, v)
	}

	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, request)

	return recorder.Result(), nil
}
