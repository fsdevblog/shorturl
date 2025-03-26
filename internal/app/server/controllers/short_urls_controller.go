package controllers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

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

func (s *ShortURLController) Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.redirect(w, r)
	case http.MethodPost:
		s.createShortURL(w, r)
	default:
		http.Error(w, ErrMethodNowAllowed.Error(), http.StatusMethodNotAllowed)
	}
}

func (s *ShortURLController) redirect(w http.ResponseWriter, r *http.Request) {
	sIdentifier := r.URL.Path[1:]

	if len(sIdentifier) != models.ShortIdentifierLength {
		http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
		return
	}

	sURL, err := s.urlService.GetByShortIdentifier(sIdentifier)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
			return
		}

		logrus.WithError(err).Error()
		http.Error(w, ErrInternal.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", sURL.URL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// CreateShortURL принимаем plain запрос со ссылкой.
func (s *ShortURLController) createShortURL(w http.ResponseWriter, r *http.Request) {
	body, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		http.Error(w, ErrInternal.Error(), http.StatusInternalServerError)
		return
	}

	parsedURL, parseErr := validateURL(string(body))

	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusUnprocessableEntity)
		return
	}

	sURL, createErr := s.urlService.Create(parsedURL.String())

	if createErr != nil {
		logrus.WithError(createErr).Error()
		http.Error(w, ErrInternal.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	_, _ = w.Write([]byte(getFullURL(r, sURL.ShortIdentifier)))
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
