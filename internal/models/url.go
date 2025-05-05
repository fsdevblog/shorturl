package models

const ShortIdentifierLength = 8

type URL struct {
	ID              uint   `json:"ID"`
	URL             string `json:"url"`
	ShortIdentifier string `json:"shortIdentifier"`
}
