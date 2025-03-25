package repositories

import "gorm.io/gorm"

type BaseRepositoryInterface interface {
	GetDB() *gorm.DB
	ExecuteTransaction(f func(tx *gorm.DB) error) error
}
