package models

import (
	"gorm.io/gorm"
)

type URL struct {
	gorm.Model
	URL             string `gorm:"index:idx_url,unique;size:255"`
	ShortIdentifier string `gorm:"index:idx_short_identifier,unique;size:8"`
}
