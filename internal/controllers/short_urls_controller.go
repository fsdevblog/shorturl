package controllers

import (
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

type URLShortener interface {
	Create(rawURL string) (*models.URL, error)
	GetByShortIdentifier(shortID string) (*models.URL, error)
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

func (s *ShortURLController) Redirect(ctx *gin.Context) {
	sIdentifier := ctx.Param("shortID")

	if len(sIdentifier) != models.ShortIdentifierLength {
		ctx.String(http.StatusNotFound, ErrRecordNotFound.Error())
		return
	}

	sURL, err := s.urlService.GetByShortIdentifier(sIdentifier)

	if err != nil {
		if errors.Is(err, services.ErrRecordNotFound) {
			ctx.String(http.StatusNotFound, err.Error())
			return
		}

		_ = ctx.Error(err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Redirect(http.StatusTemporaryRedirect, sURL.URL)
}

type createParams struct {
	URL string `json:"url"`
}

// CreateShortURL создает ссылку.
func (s *ShortURLController) CreateShortURL(ctx *gin.Context) {
	strongParams, err := s.bindCreateParams(ctx)
	if err != nil {
		ctx.String(http.StatusUnprocessableEntity, err.Error())
		return
	}

	parsedURL, parseErr := validateURL(strongParams.URL)

	if parseErr != nil {
		ctx.String(http.StatusUnprocessableEntity, parseErr.Error())
		return
	}

	sURL, createErr := s.urlService.Create(parsedURL.String())

	if createErr != nil {
		ctx.String(http.StatusInternalServerError, createErr.Error())
		return
	}

	if isJSONRequest(ctx) {
		ctx.JSON(http.StatusCreated, gin.H{"result": s.getShortURL(ctx.Request, sURL.ShortIdentifier)})
	} else {
		ctx.String(http.StatusCreated, s.getShortURL(ctx.Request, sURL.ShortIdentifier))
	}
}

// bindCreateParams байндит json или application/x-www-form-urlencoded запросы для создания ссылки.
func (s *ShortURLController) bindCreateParams(ctx *gin.Context) (*createParams, error) {
	var params createParams
	body, readErr := io.ReadAll(ctx.Request.Body)
	if readErr != nil {
		_ = ctx.Error(fmt.Errorf("bind params: %w", readErr))
		return nil, ErrInternal
	}

	if !isJSONRequest(ctx) {
		params.URL = string(body)
	} else {
		if jsonErr := json.Unmarshal(body, &params); jsonErr != nil {
			_ = ctx.Error(fmt.Errorf("bind params: %w", jsonErr))
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
