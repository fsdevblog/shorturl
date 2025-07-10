package models

import "time"

// ShortIdentifierLength длина короткой ссылки.
const ShortIdentifierLength = 8

// URL структура модели хранения URL.
type URL struct {
	ID              uint       `json:"ID"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       *time.Time `json:"deletedAt"`
	URL             string     `json:"url"`
	ShortIdentifier string     `json:"shortIdentifier"`
	VisitorUUID     string     `json:"visitorUUID"`
}
