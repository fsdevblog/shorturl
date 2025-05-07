package services

import "github.com/fsdevblog/shorturl/internal/models"

type BatchResponseItem[T any] struct {
	Item T
	Err  error
}
type BatchExecResponse[T any] struct {
	results []BatchResponseItem[T]
}

func NewBatchExecResponse[T any](allocSize int) *BatchExecResponse[T] {
	return &BatchExecResponse[T]{
		results: make([]BatchResponseItem[T], allocSize),
	}
}

// Set вставляет в срез данные типа T по индексу index.
// Использовать желательно только в тестах. Потоконебезопасно.
func (b *BatchExecResponse[T]) Set(r BatchResponseItem[T], index int) {
	b.results[index] = r
}

func (b *BatchExecResponse[T]) Len() int {
	return len(b.results)
}

func (b *BatchExecResponse[T]) ReadResponse(fn func(int, T, error)) {
	if fn == nil {
		return
	}
	for i, result := range b.results {
		fn(i, result.Item, result.Err)
	}
}

type BatchCreateShortURLsResponse struct {
	*BatchExecResponse[models.URL]
}

func NewBatchExecResponseURL(inner *BatchExecResponse[models.URL]) *BatchCreateShortURLsResponse {
	return &BatchCreateShortURLsResponse{inner}
}
