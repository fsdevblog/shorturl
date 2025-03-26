package controllers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"

	"github.com/fsdevblog/shorturl/internal/app/models"

	"github.com/sirupsen/logrus"

	"github.com/fsdevblog/shorturl/internal/app/services"
)

// hostnameRegex в соответствии с `RFC 1123` за исключением - исключает корневые доменные имена (без зоны).
var hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9](-?[a-zA-Z0-9])*\.)+([a-zA-Z0-9](-?[a-zA-Z0-9])*)$`)

type ShortURLController struct {
	urlService services.IURLService
}

func NewShortURLController(urlService services.IURLService) *ShortURLController {
	return &ShortURLController{
		urlService: urlService,
	}
}

func (s *ShortURLController) Redirect(ctx *gin.Context) {
	sIdentifier := ctx.Param("shortID")

	if len(sIdentifier) != models.ShortIdentifierLength {
		ctx.String(http.StatusNotFound, ErrNotFound.Error())
		return
	}

	sURL, err := s.urlService.GetByShortIdentifier(sIdentifier)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.String(http.StatusNotFound, ErrNotFound.Error())
			return
		}

		logrus.WithError(err).Error()
		ctx.String(http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	ctx.Redirect(http.StatusTemporaryRedirect, sURL.URL)
}

// CreateShortURL принимаем plain запрос со ссылкой.
func (s *ShortURLController) CreateShortURL(ctx *gin.Context) {
	body, readErr := io.ReadAll(ctx.Request.Body)
	if readErr != nil {
		ctx.String(http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	parsedURL, parseErr := validateURL(string(body))

	if parseErr != nil {
		ctx.String(http.StatusUnprocessableEntity, parseErr.Error())
		return
	}

	sURL, createErr := s.urlService.Create(parsedURL.String())

	if createErr != nil {
		logrus.WithError(createErr).Error()
		ctx.String(http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	ctx.String(http.StatusCreated, getFullURL(ctx.Request, sURL.ShortIdentifier))
}

// getFullURL создает короткую ссылку.
func getFullURL(r *http.Request, shortID string) string {
	var scheme = "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, r.Host, shortID)
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
