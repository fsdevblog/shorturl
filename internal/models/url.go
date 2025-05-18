package models

import "time"

const ShortIdentifierLength = 8

type URL struct {
	ID              uint      `json:"ID"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	URL             string    `json:"url"`
	ShortIdentifier string    `json:"shortIdentifier"`
	VisitorUUID     *string   `json:"visitorUUID"`
}
