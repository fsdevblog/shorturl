package models

const ShortIdentifierLength = 8

type URL struct {
	ID              uint   `gorm:"primary_key"                              json:"ID"`
	URL             string `gorm:"index:idx_url,unique;size:255"            json:"url"`
	ShortIdentifier string `gorm:"index:idx_short_identifier,unique;size:8" json:"shortIdentifier"`
}
