package dto

// UserURLResponse DTO объект для UserURLs.
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// BatchCreateParams параметры для пакетного создания URL.
type BatchCreateParams struct {
	// CorrelationID уникальный идентификатор для корреляции запроса
	CorrelationID string `json:"correlation_id"`
	// OriginalURL исходный URL для сокращения
	OriginalURL string `json:"original_url"`
}

// BatchCreateResponse данные, возвращаемые при массовой вставке.
type BatchCreateResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url,omitempty"`
	Error         error  `json:"error"`
}
