package models

const ShortIdentifierLength = 8

type URL struct {
	ID              uint   `gorm:"primary_key"`
	URL             string `gorm:"index:idx_url,unique;size:255"`
	ShortIdentifier string `gorm:"index:idx_short_identifier,unique;size:8"`
}
