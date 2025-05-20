package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/fsdevblog/shorturl/internal/controllers/middlewares"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/services"

	"github.com/gin-gonic/gin"
)

// hostnameRegex в соответствии с `RFC 1123` за исключением - исключает корневые доменные имена (без зоны).
var hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9](-?[a-zA-Z0-9])*\.)+([a-zA-Z0-9](-?[a-zA-Z0-9])*)$`)

type ShortURLController struct {
	urlService ShortURLStore
	baseURL    *url.URL
}

func NewShortURLController(urlService ShortURLStore, baseURL *url.URL) *ShortURLController {
	return &ShortURLController{
		urlService: urlService,
		baseURL:    baseURL,
	}
}

type BatchCreateParams struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}
type BatchCreateResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url,omitempty"`
}

type URLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (s *ShortURLController) UserURLs(c *gin.Context) {
	vu, _ := c.Get(middlewares.VisitorUUIDKey)
	visitorUUID, vOK := vu.(string)
	if !vOK {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()

	urls, err := s.urlService.GetAllByVisitorUUID(ctx, visitorUUID)
	if err != nil {
		_ = c.Error(fmt.Errorf("get user urls: %w", err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}

	var r = make([]URLResponse, len(urls))
	for i, u := range urls {
		r[i] = URLResponse{
			ShortURL:    s.getShortURL(c.Request, u.ShortIdentifier),
			OriginalURL: u.URL,
		}
	}
	c.JSON(http.StatusOK, r)
}

func (s *ShortURLController) BatchCreate(c *gin.Context) {
	vu, _ := c.Get(middlewares.VisitorUUIDKey)
	visitorUUID, vOK := vu.(*string)
	if !vOK {
		visitorUUID = nil
	}

	var params []BatchCreateParams
	if bindErr := c.ShouldBindJSON(&params); bindErr != nil {
		_ = c.Error(fmt.Errorf("bind params: %w", bindErr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request. Only json is supported"})
		return
	}

	var urlMap = make(map[string]string, len(params))
	var rawURLs = make([]string, len(params))

	for i, param := range params {
		_, parseErr := validateURL(param.OriginalURL)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":          param.OriginalURL + " is invalid URL",
				"correlation_id": param.CorrelationID,
			})
			return
		}
		urlMap[param.OriginalURL] = param.CorrelationID
		rawURLs[i] = param.OriginalURL
	}

	if len(rawURLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request"})
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()

	batchResponse, err := s.urlService.BatchCreate(ctx, visitorUUID, rawURLs)
	if err != nil {
		_ = c.Error(fmt.Errorf("batch create urls: %w", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": ErrInternal.Error()})
		return
	}

	var response = make([]BatchCreateResponse, batchResponse.Len())
	var statusCode = http.StatusCreated

	batchResponse.ReadResponse(func(i int, m models.URL, err error) {
		cid, ok := urlMap[m.URL]
		var tmpErrs []error
		if !ok {
			cid = ""
			errMsg := "correlation id not found for url: " + m.URL
			tmpErrs = append(tmpErrs, errors.New(errMsg))
		}
		if err != nil {
			if errors.Is(err, services.ErrDuplicateKey) {
				statusCode = http.StatusConflict
			} else {
				tmpErrs = append(tmpErrs, err)
			}
		}
		if len(tmpErrs) > 0 {
			_ = c.Error(errors.Join(tmpErrs...))
		}

		response[i] = BatchCreateResponse{
			CorrelationID: cid,
			ShortURL:      s.getShortURL(c.Request, m.ShortIdentifier),
		}
	})

	c.JSON(statusCode, response)
}

func (s *ShortURLController) Redirect(c *gin.Context) {
	sIdentifier := c.Param("shortID")

	if len(sIdentifier) != models.ShortIdentifierLength {
		c.String(http.StatusNotFound, ErrRecordNotFound.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()

	sURL, err := s.urlService.GetByShortIdentifier(ctx, sIdentifier)

	if err != nil {
		if errors.Is(err, services.ErrRecordNotFound) {
			c.String(http.StatusNotFound, err.Error())
			return
		}

		_ = c.Error(err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if sURL.DeletedAt != nil {
		c.AbortWithStatus(http.StatusGone)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, sURL.URL)
}

type createParams struct {
	URL string `json:"url"`
}

// CreateShortURL создает ссылку.
func (s *ShortURLController) CreateShortURL(c *gin.Context) {
	vu, _ := c.Get(middlewares.VisitorUUIDKey)
	visitorUUID, ok := vu.(string)
	if !ok {
		_ = c.Error(errors.New("visitor cookie not found"))
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	strongParams, err := s.bindCreateParams(c)
	if err != nil {
		c.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	parsedURL, parseErr := validateURL(strongParams.URL)

	if parseErr != nil {
		c.String(http.StatusUnprocessableEntity, parseErr.Error())
		return
	}

	sURL, isNewRecord, createErr := s.urlService.Create(c, &visitorUUID, parsedURL.String())
	if createErr != nil {
		_ = c.Error(createErr)
		c.String(http.StatusInternalServerError, createErr.Error())
		return
	}

	var statusCode = http.StatusCreated
	if !isNewRecord {
		statusCode = http.StatusConflict
	}

	if isJSONRequest(c) {
		c.JSON(statusCode, gin.H{"result": s.getShortURL(c.Request, sURL.ShortIdentifier)})
	} else {
		c.String(statusCode, s.getShortURL(c.Request, sURL.ShortIdentifier))
	}
}

// DeleteUserURLs DELETE /api/user/urls.
func (s *ShortURLController) DeleteUserURLs(c *gin.Context) {
	var ids []string
	if bindErr := c.ShouldBindJSON(&ids); bindErr != nil || len(ids) == 0 {
		_ = c.Error(fmt.Errorf("bind params: %w", bindErr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request. Only json is supported"})
		return
	}

	vu, _ := c.Get(middlewares.VisitorUUIDKey)
	visitorUUID, ok := vu.(string)
	if !ok {
		_ = c.Error(errors.New("visitor cookie not found"))
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()
	if err := s.urlService.MarkAsDeleted(ctx, ids, visitorUUID); err != nil {
		_ = c.Error(fmt.Errorf("delete user urls: %w", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": ErrInternal.Error()})
		return
	}
	c.Status(http.StatusAccepted)
}

// bindCreateParams байндит json или application/x-www-form-urlencoded запросы для создания ссылки.
func (s *ShortURLController) bindCreateParams(c *gin.Context) (*createParams, error) {
	var params createParams
	body, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		_ = c.Error(fmt.Errorf("bind params: %w", readErr))
		return nil, ErrInternal
	}

	if !isJSONRequest(c) {
		params.URL = string(body)
	} else {
		if jsonErr := json.Unmarshal(body, &params); jsonErr != nil {
			_ = c.Error(fmt.Errorf("bind params: %w", jsonErr))
			return nil, ErrInternal
		}
	}
	return &params, nil
}

// getShortURL вспомогательный метод который создает короткую ссылку.
func (s *ShortURLController) getShortURL(r *http.Request, shortID string) string {
	var scheme = "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if s.baseURL == nil {
		return fmt.Sprintf("%s://%s/%s", scheme, r.Host, shortID)
	}
	return fmt.Sprintf("%s/%s", s.baseURL, shortID)
}

// validateURL проверяет, является ли строка корректным URL.
func validateURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)

	if err != nil {
		return nil, errors.New("invalid URL format")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("URL must have http or https scheme")
	}

	if parsedURL.Host == "" {
		return nil, errors.New("URL must have a host")
	}

	if parsedURL.Hostname() != "localhost" && !hostnameRegex.MatchString(parsedURL.Hostname()) {
		return nil, errors.New("invalid hostname")
	}

	return parsedURL, nil
}
