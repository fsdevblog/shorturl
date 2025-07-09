package repositories

import "github.com/fsdevblog/shorturl/internal/models"

type BatchResult[T any] struct {
	Value T
	Err   error
}

type BatchCreateArg struct {
	ShortIdentifier string
	URL             string
	VisitorUUID     string
}

type BatchCreateShortURLsResult struct {
	Results []BatchResult[models.URL]
}
