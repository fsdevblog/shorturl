package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/fsdevblog/shorturl/internal/transport/http/middlewares"
	"github.com/fsdevblog/shorturl/internal/transport/trnptf"
	fdto "github.com/fsdevblog/shorturl/internal/transport/trnptf/dto"

	"github.com/fsdevblog/shorturl/internal/services"

	"github.com/gin-gonic/gin"
)

// ShortURLController обрабатывает HTTP запросы для работы с короткими URL.
// Предоставляет методы для создания, получения и управления короткими URL.
type ShortURLController struct {
	urlFacade *trnptf.URL
}

// NewShortURLController создает новый экземпляр ShortURLController.
//
// Параметры:
//   - urlService: сервис для работы с URL
//   - baseURL: базовый URL для генерации коротких ссылок
//
// Возвращает:
//   - *ShortURLController: новый экземпляр контроллера
func NewShortURLController(urlFacade *trnptf.URL) *ShortURLController {
	return &ShortURLController{
		urlFacade: urlFacade,
	}
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

	urls, err := s.urlFacade.UserURLs(ctx, visitorUUID)
	if err != nil {
		_ = c.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, urls)
}

// BatchCreate создает несколько коротких URL одновременно.
// Принимает массив BatchCreateParams в формате JSON.
//
// Коды ответа:
//   - 201: URL успешно созданы
//   - 400: некорректный запрос
//   - 401: пользователь не авторизован
//   - 500: внутренняя ошибка сервера
func (s *ShortURLController) BatchCreate(c *gin.Context) {
	vu, _ := c.Get(middlewares.VisitorUUIDKey)
	visitorUUID, vOK := vu.(string)
	if !vOK {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var params []fdto.BatchCreateParams
	if bindErr := c.ShouldBindJSON(&params); bindErr != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if len(params) == 0 {
		_ = c.AbortWithError(http.StatusBadRequest, errors.New("empty request")).
			SetType(gin.ErrorTypePublic)
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()
	createResponse, errCreate := s.urlFacade.BatchCreate(ctx, visitorUUID, params)

	if errCreate != nil {
		_ = c.Error(fmt.Errorf("batch create: %w", errCreate))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, createResponse)
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

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()

	urlToRedirect, err := s.urlFacade.GetOriginalURL(ctx, sIdentifier)

	if err != nil {
		if errors.Is(err, services.ErrRecordNotFound) {
			c.String(http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, trnptf.ErrBadRequest) {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		if errors.Is(err, trnptf.ErrRecordDeleted) {
			c.AbortWithStatus(http.StatusGone)
			return
		}

		_ = c.Error(err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, urlToRedirect)
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

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()
	sURL, isNewRecord, errCreate := s.urlFacade.CreateShortURL(ctx, visitorUUID, strongParams.URL)

	if errCreate != nil {
		if errors.Is(errCreate, trnptf.ErrInvalidURL) {
			_ = c.AbortWithError(http.StatusUnprocessableEntity, errCreate).
				SetType(gin.ErrorTypePublic)
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, errCreate).
			SetType(gin.ErrorTypePrivate)
		return
	}

	var statusCode = http.StatusCreated
	if !isNewRecord {
		statusCode = http.StatusConflict
	}

	if isJSONRequest(c) {
		c.JSON(statusCode, gin.H{"result": sURL})
	} else {
		c.String(statusCode, sURL)
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
		_ = c.AbortWithError(http.StatusForbidden, errors.New("visitor cookie not found")).
			SetType(gin.ErrorTypePublic)
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultRequestTimeout)
	defer cancel()
	if err := s.urlFacade.DeleteUserURLs(ctx, visitorUUID, ids); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("delete user urls: %w", err)).
			SetType(gin.ErrorTypePrivate)
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
