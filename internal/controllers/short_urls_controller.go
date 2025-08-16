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

// hostnameRegex регулярное выражение для проверки hostname в соответствии с RFC 1123.
// Исключает корневые доменные имена (без зоны).
var hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9](-?[a-zA-Z0-9])*\.)+([a-zA-Z0-9](-?[a-zA-Z0-9])*)$`)

// ShortURLController обрабатывает HTTP запросы для работы с короткими URL.
// Предоставляет методы для создания, получения и управления короткими URL.
type ShortURLController struct {
	urlService ShortURLStore
	baseURL    string
}

// NewShortURLController создает новый экземпляр ShortURLController.
//
// Параметры:
//   - urlService: сервис для работы с URL
//   - baseURL: базовый URL для генерации коротких ссылок
//
// Возвращает:
//   - *ShortURLController: новый экземпляр контроллера
func NewShortURLController(urlService ShortURLStore, baseURL string) *ShortURLController {
	return &ShortURLController{
		urlService: urlService,
		baseURL:    baseURL,
	}
}

// BatchCreateParams параметры для пакетного создания URL.
type BatchCreateParams struct {
	// CorrelationID уникальный идентификатор для корреляции запроса
	CorrelationID string `json:"correlation_id"`
	// OriginalURL исходный URL для сокращения
	OriginalURL string `json:"original_url"`
}

// BatchCreateResponse ответ на пакетное создание URL.
type BatchCreateResponse struct {
	// CorrelationID идентификатор соответствующего запроса
	CorrelationID string `json:"correlation_id"`
	// ShortURL сгенерированный короткий URL
	ShortURL string `json:"short_url,omitempty"`
}

// URLResponse структура ответа.
type URLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// UserURLs возвращает список всех URL, созданных текущим пользователем.
// Требует наличия VisitorUUID в контексте запроса.
//
// Коды ответа:
//   - 200: успешное получение списка URL
//   - 204: у пользователя нет созданных URL
//   - 403: отсутствует или недействителен VisitorUUID
//   - 500: внутренняя ошибка сервера
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

// BatchCreate создает несколько коротких URL одновременно.
// Принимает массив BatchCreateParams в формате JSON.
//
// Коды ответа:
//   - 201: URL успешно созданы
//   - 400: некорректный запрос
//   - 401: пользователь не авторизован
//   - 409: обнаружен конфликт (дубликат URL)
//   - 500: внутренняя ошибка сервера
func (s *ShortURLController) BatchCreate(c *gin.Context) {
	vu, _ := c.Get(middlewares.VisitorUUIDKey)
	visitorUUID, vOK := vu.(string)
	if !vOK {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
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

// Redirect выполняет перенаправление с короткого URL на оригинальный.
//
// Параметры URL:
//   - shortID: короткий идентификатор URL
//
// Коды ответа:
//   - 307: временное перенаправление
//   - 404: URL не найден
//   - 410: URL был удален
//   - 500: внутренняя ошибка сервера
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

// CreateShortURL создает новый короткий URL.
// Принимает URL в формате JSON или plain text.
//
// Коды ответа:
//   - 201: URL успешно создан
//   - 409: URL уже существует
//   - 422: некорректный URL
//   - 401: пользователь не авторизован
//   - 500: внутренняя ошибка сервера
func (s *ShortURLController) CreateShortURL(c *gin.Context) {
	vu, _ := c.Get(middlewares.VisitorUUIDKey)
	visitorUUID, ok := vu.(string)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
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

	sURL, isNewRecord, createErr := s.urlService.Create(c, visitorUUID, parsedURL.String())
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

// DeleteUserURLs удаляет URL пользователя.
// Принимает массив идентификаторов URL в формате JSON.
//
// Коды ответа:
//   - 202: запрос на удаление принят
//   - 400: некорректный запрос
//   - 403: доступ запрещен
//   - 500: внутренняя ошибка сервера
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

// bindCreateParams байндит параметры создания URL из запроса.
// Поддерживает форматы JSON и application/x-www-form-urlencoded.
//
// Возвращает:
//   - *createParams: параметры создания URL
//   - error: ошибка при обработке запроса
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

// getShortURL формирует полный короткий URL на основе идентификатора.
//
// Параметры:
//   - r: HTTP запрос
//   - shortID: короткий идентификатор URL
//
// Возвращает:
//   - string: полный короткий URL
func (s *ShortURLController) getShortURL(r *http.Request, shortID string) string {
	var scheme = "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if s.baseURL == "" {
		return fmt.Sprintf("%s://%s/%s", scheme, r.Host, shortID)
	}
	return fmt.Sprintf("%s/%s", s.baseURL, shortID)
}

// validateURL проверяет корректность URL.
//
// Параметры:
//   - rawURL: URL для проверки
//
// Возвращает:
//   - *url.URL: распарсенный URL
//   - error: ошибка валидации
//
// Правила валидации:
//   - URL должен иметь схему http или https
//   - URL должен содержать хост
//   - Hostname должен соответствовать RFC 1123 или быть localhost
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
