package utils

import (
	"crypto/md5" //nolint:gosec
	"encoding/base64"
)

// GenerateShortID генерирует идентификатор для ссылки нужной длины на основе delta.
func GenerateShortID(rawURL string, delta uint, length int) string {
	// Добавляем счетчик к срезу (для избежания коллизий)
	b := []byte(rawURL)
	b = append(b, byte(delta))

	// Создаем хеш и конвертим в base62
	hash := md5.Sum(b) //nolint:gosec
	base62 := base64.URLEncoding.EncodeToString(hash[:])
	return base62[:length]
}
