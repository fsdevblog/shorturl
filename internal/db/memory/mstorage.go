package memory

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"
)

type MStorage struct {
	data map[string][]byte
	m    sync.RWMutex
}

func NewMemStorage() *MStorage {
	return &MStorage{
		data: make(map[string][]byte),
	}
}

func (m *MStorage) Len() int {
	return len(m.data)
}

func (m *MStorage) IsExist(key string) bool {
	m.m.RLock()
	defer m.m.RUnlock()

	_, ok := m.data[key]
	return ok
}

func Get[T any](key string, m *MStorage) (*T, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	val, ok := m.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	var result T
	if err := json.Unmarshal(val, &result); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal json by key `%s`", key)
	}
	return &result, nil
}

// Set Сохраняет новые пары ключ/значение. Ключ обязан быть уникальным, иначе вернется ошибка ErrDuplicateKey.
func Set[T any](key string, val *T, m *MStorage) error {
	if m.IsExist(key) {
		return ErrDuplicateKey
	}
	m.m.Lock()
	defer m.m.Unlock()

	bytes, err := json.Marshal(val)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal json for object `%+v`", val)
	}
	m.data[key] = bytes
	return nil
}

func GetAll[T any](m *MStorage) []T {
	m.m.RLock()
	defer m.m.RUnlock()

	var result = make([]T, 0, len(m.data))

	for _, bytes := range m.data {
		var val T
		if err := json.Unmarshal(bytes, &val); err != nil {
			logrus.WithError(err).Errorf("failed to unmarshal json for object `%+v`", val)
			continue
		}
		result = append(result, val)
	}
	return result
}
