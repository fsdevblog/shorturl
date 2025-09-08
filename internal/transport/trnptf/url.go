package trnptf

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/transport/trnptf/dto"
)

// hostnameRegex регулярное выражение для проверки hostname в соответствии с RFC 1123.
// Исключает корневые доменные имена (без зоны).
var hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9](-?[a-zA-Z0-9])*\.)+([a-zA-Z0-9](-?[a-zA-Z0-9])*)$`)

// URL представляет собой структуру для работы с URL-адресами.
// Содержит конфигурацию и зависимости для обработки URL.
type URL struct {
	scheme   string
	host     string
	provider URLProvider // Сервис для работы со ссылками.
	baseURL  string
}

// New создает новый экземпляр URL с заданными параметрами.
//
// Параметры:
//   - isHTTPS: флаг использования HTTPS протокола
//   - host: хост сервера
//   - baseURL: базовый URL для формирования коротких ссылок
//   - provider: провайдер для работы с URL
//
// Возвращает:
//   - *URL: новый экземпляр структуры URL
func New(isHTTPS bool, host string, baseURL string, provider URLProvider) *URL {
	var scheme = "http"
	if isHTTPS {
		scheme = "https"
	}
	return &URL{
		scheme:   scheme,
		host:     host,
		provider: provider,
		baseURL:  baseURL,
	}
}

// UserURLs возвращает все URL-адреса, связанные с определенным посетителем.
//
// Параметры:
//   - ctx: контекст выполнения
//   - visitorUUID: уникальный идентификатор посетителя
//
// Возвращает:
//   - []dto.UserURLResponse: список URL-адресов пользователя
//   - error: ошибка при выполнении операции
func (u *URL) UserURLs(ctx context.Context, visitorUUID string) ([]dto.UserURLResponse, error) {
	urls, err := u.provider.GetAllByVisitorUUID(ctx, visitorUUID)
	if err != nil {
		return nil, fmt.Errorf("get user urls: %w", err)
	}
	var r = make([]dto.UserURLResponse, len(urls))
	for i, ur := range urls {
		r[i] = dto.UserURLResponse{
			ShortURL:    u.calcShortURL(ur.ShortIdentifier),
			OriginalURL: ur.URL,
		}
	}
	return r, nil
}

// BatchCreate создает несколько коротких URL-адресов одновременно.
//
// Параметры:
//   - ctx: контекст выполнения
//   - visitorUUID: уникальный идентификатор посетителя
//   - params: параметры для создания URL-адресов
//
// Возвращает:
//   - []dto.BatchCreateResponse: результаты создания URL-адресов
//   - error: ошибка при выполнении операции
func (u *URL) BatchCreate(
	ctx context.Context,
	visitorUUID string,
	params []dto.BatchCreateParams,
) ([]dto.BatchCreateResponse, error) {
	var urlMap = make(map[string]string, len(params))
	var rawURLs = make([]string, len(params))

	for i, param := range params {
		_, parseErr := validateURL(param.OriginalURL)
		if parseErr != nil {
			return nil, NewInvalidURLInBatch(param.CorrelationID, param.OriginalURL)
		}
		urlMap[param.OriginalURL] = param.CorrelationID
		rawURLs[i] = param.OriginalURL
	}

	result, err := u.provider.BatchCreate(ctx, visitorUUID, rawURLs)
	if err != nil {
		return nil, fmt.Errorf("batch create urls: %w", err)
	}

	var response = make([]dto.BatchCreateResponse, result.Len())
	result.ReadResponse(func(i int, m models.URL, err error) {
		response[i] = dto.BatchCreateResponse{
			ShortURL: u.calcShortURL(m.ShortIdentifier),
		}
		cid, ok := urlMap[m.URL]
		if !ok {
			response[i].Error = errors.New("correlation id not found")
		}
		if err != nil {
			response[i].Error = errors.Join(response[i].Error, err)
		}
		response[i].CorrelationID = cid
	})

	return response, nil
}

// GetOriginalURL получает оригинальный URL-адрес по короткому идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения
//   - shortID: короткий идентификатор URL
//
// Возвращает:
//   - string: оригинальный URL-адрес
//   - error: ошибка при выполнении операции
func (u *URL) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	if len(shortID) != models.ShortIdentifierLength {
		return "", ErrBadRequest
	}
	sURL, err := u.provider.GetByShortIdentifier(ctx, shortID)
	if err != nil {
		return "", fmt.Errorf("get short url: %w", err)
	}
	if sURL.DeletedAt != nil {
		return "", ErrRecordDeleted
	}

	return sURL.URL, nil
}

// CreateShortURL создает новый короткий URL-адрес.
//
// Параметры:
//   - ctx: контекст выполнения
//   - visitorUUID: уникальный идентификатор посетителя
//   - url: оригинальный URL-адрес
//
// Возвращает:
//   - string: короткий URL-адрес
//   - bool: флаг, указывающий является ли запись новой
//   - error: ошибка при выполнении операции
func (u *URL) CreateShortURL(ctx context.Context, visitorUUID string, url string) (string, bool, error) {
	parsedURL, errValidate := validateURL(url)
	if errValidate != nil {
		return "", false, ErrInvalidURL
	}
	sURL, isNewRecord, errCreate := u.provider.Create(ctx, visitorUUID, parsedURL.String())
	if errCreate != nil {
		return "", false, fmt.Errorf("create short url: %w", errCreate)
	}
	return u.calcShortURL(sURL.ShortIdentifier), isNewRecord, nil
}

// DeleteUserURLs помечает URL-адреса пользователя как удаленные.
//
// Параметры:
//   - ctx: контекст выполнения
//   - visitorUUID: уникальный идентификатор посетителя
//   - ids: список идентификаторов URL для удаления
//
// Возвращает:
//   - error: ошибка при выполнении операции
func (u *URL) DeleteUserURLs(ctx context.Context, visitorUUID string, ids []string) error {
	if err := u.provider.MarkAsDeleted(ctx, ids, visitorUUID); err != nil {
		return fmt.Errorf("mark as deleted: %w", err)
	}
	return nil
}

// calcShortURL формирует полный короткий URL на основе идентификатора.
//
// Параметры:
//   - shortID: короткий идентификатор URL
//
// Возвращает:
//   - string: полный короткий URL
func (u *URL) calcShortURL(shortID string) string {
	if u.baseURL == "" {
		return fmt.Sprintf("%s://%s/%s", u.scheme, u.host, shortID)
	}
	return fmt.Sprintf("%s/%s", u.baseURL, shortID)
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
