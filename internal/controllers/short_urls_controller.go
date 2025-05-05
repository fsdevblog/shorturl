package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/pkg/errors"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/services"

	"github.com/gin-gonic/gin"
)

//go:generate mockgen -source=short_urls_controller.go -destination=mocksctrl/url_shortener.go -package=mocksctrl
type URLShortener interface {
	Create(ctx context.Context, rawURL string) (*models.URL, error)
	GetByShortIdentifier(ctx context.Context, shortID string) (*models.URL, error)
}

// hostnameRegex в соответствии с `RFC 1123` за исключением - исключает корневые доменные имена (без зоны).
var hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9](-?[a-zA-Z0-9])*\.)+([a-zA-Z0-9](-?[a-zA-Z0-9])*)$`)

type ShortURLController struct {
	urlService URLShortener
	baseURL    *url.URL
}

func NewShortURLController(urlService URLShortener, baseURL *url.URL) *ShortURLController {
	return &ShortURLController{
		urlService: urlService,
		baseURL:    baseURL,
	}
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

	c.Redirect(http.StatusTemporaryRedirect, sURL.URL)
}

type createParams struct {
	URL string `json:"url"`
}

// CreateShortURL создает ссылку.
func (s *ShortURLController) CreateShortURL(c *gin.Context) {
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

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()

	sURL, createErr := s.urlService.Create(ctx, parsedURL.String())

	if createErr != nil {
		c.String(http.StatusInternalServerError, createErr.Error())
		return
	}

	if isJSONRequest(c) {
		c.JSON(http.StatusCreated, gin.H{"result": s.getShortURL(c.Request, sURL.ShortIdentifier)})
	} else {
		c.String(http.StatusCreated, s.getShortURL(c.Request, sURL.ShortIdentifier))
	}
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
