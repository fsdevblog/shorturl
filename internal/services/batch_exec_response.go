package services

import "github.com/fsdevblog/shorturl/internal/models"

// BatchResponseItem содержит результат выполнения одной операции в пакете.
//
// Параметры типа:
//   - T: тип данных операции
type BatchResponseItem[T any] struct {
	Item T
	Err  error
}

// BatchExecResponse представляет результаты выполнения пакета операций.
//
// Параметры типа:
//   - T: тип данных операций
type BatchExecResponse[T any] struct {
	results []BatchResponseItem[T]
}

// NewBatchExecResponse создает новый экземпляр BatchExecResponse с предварительно выделенной памятью.
//
// Параметры:
//   - allocSize: размер для предварительного выделения памяти
//
// Возвращает:
//   - *BatchExecResponse[T]: инициализированный экземпляр
func NewBatchExecResponse[T any](allocSize int) *BatchExecResponse[T] {
	return &BatchExecResponse[T]{
		results: make([]BatchResponseItem[T], allocSize),
	}
}

// Set устанавливает результат операции по указанному индексу.
// Примечание: метод не является потокобезопасным.
//
// Параметры:
//   - r: результат операции
//   - index: индекс для вставки
func (b *BatchExecResponse[T]) Set(r BatchResponseItem[T], index int) {
	b.results[index] = r
}

// Len возвращает количество операций в пакете.
//
// Возвращает:
//   - int: количество операций
func (b *BatchExecResponse[T]) Len() int {
	return len(b.results)
}

// ReadResponse обрабатывает результаты всех операций в пакете.
//
// Параметры:
//   - fn: функция обработки результата. Принимает:
//   - индекс операции
//   - результат операции типа T
//   - ошибку операции
func (b *BatchExecResponse[T]) ReadResponse(fn func(int, T, error)) {
	if fn == nil {
		return
	}
	for i, result := range b.results {
		fn(i, result.Item, result.Err)
	}
}

// BatchCreateShortURLsResponse специализированный тип BatchExecResponse для операций с URL.
// Данная обертка необходима для работы mockgen, т.е. он не умеет работать с дженериками.
type BatchCreateShortURLsResponse struct {
	*BatchExecResponse[models.URL]
}

// NewBatchExecResponseURL создает новый экземпляр BatchCreateShortURLsResponse.
//
// Параметры:
//   - inner: базовый BatchExecResponse для URL
//
// Возвращает:
//   - *BatchCreateShortURLsResponse: специализированный экземпляр для URL
func NewBatchExecResponseURL(inner *BatchExecResponse[models.URL]) *BatchCreateShortURLsResponse {
	return &BatchCreateShortURLsResponse{inner}
}
