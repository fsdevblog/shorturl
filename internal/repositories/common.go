package repositories

import "github.com/fsdevblog/shorturl/internal/models"

// BatchResult представляет результат операции для одного элемента в пакетной обработке.
//
// Параметры типа:
//   - T: тип значения результата
type BatchResult[T any] struct {
	Value T
	Err   error
}

// BatchCreateArg содержит данные для создания короткого URL.
type BatchCreateArg struct {
	ShortIdentifier string // Короткий идентификатор URL
	URL             string // Оригинальный URL
	VisitorUUID     string // Идентификатор посетителя
}

// BatchCreateShortURLsResult содержит результаты пакетного создания коротких URL.
type BatchCreateShortURLsResult struct {
	Results []BatchResult[models.URL] // Результаты для каждого URL
}
