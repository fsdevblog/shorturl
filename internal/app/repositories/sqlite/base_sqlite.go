package sqlite

import "gorm.io/gorm"

type BaseRepository struct {
	db *gorm.DB
}

func (r *BaseRepository) GetDB() *gorm.DB {
	return r.db
}

func (r *BaseRepository) ExecuteTransaction(f func(tx *gorm.DB) error) error {
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
