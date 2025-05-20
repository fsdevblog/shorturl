package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/goccy/go-json"
)

// MStorage реализация хранилища в памяти.
// Контекст ей по идее не нужен, но для того чтоб не писать доп обертку в сервисах чтоб
// все соответствовало нужному интерфейсу, будем считать что контекст жизненно необходим
// для предотвращения длительных блокировок и тд и тп.
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

// Ping бесполезный метод. Заглушка, всегда возвращает nil.
func (m *MStorage) Ping(_ context.Context) error {
	return nil
}

func (m *MStorage) IsExist(ctx context.Context, key string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err() //nolint:wrapcheck
	default:
		m.m.RLock()
		defer m.m.RUnlock()

		_, ok := m.data[key]
		return ok, nil
	}
}

func Get[T any](ctx context.Context, key string, m *MStorage) (*T, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err() //nolint:wrapcheck
	default:
		m.m.RLock()
		defer m.m.RUnlock()

		val, ok := m.data[key]
		if !ok {
			return nil, ErrNotFound
		}
		var result T
		if err := json.Unmarshal(val, &result); err != nil {
			return nil, fmt.Errorf("unmarshal by Key %s: %w", key, err)
		}
		return &result, nil
	}
}

type SetOptions struct {
	Overwrite bool
}

func WithOverwrite() func(*SetOptions) {
	return func(o *SetOptions) {
		o.Overwrite = true
	}
}

type BatchResult struct {
	Key string
	Err error
}

func BatchSet[T any](ctx context.Context, values map[string]*T, m *MStorage, opts ...func(*SetOptions)) []BatchResult {
	var br = make([]BatchResult, len(values))
	i := 0
	for key, val := range values {
		err := Set(ctx, key, val, m, opts...)
		br[i] = BatchResult{Key: key, Err: err}
		i++
	}
	return br
}

// то при попытке добавить уже существующий ключ будет возвращена ошибка ErrDuplicateKey.
func Set[T any](ctx context.Context, key string, val *T, m *MStorage, opts ...func(*SetOptions)) error {
	options := &SetOptions{Overwrite: false}
	for _, opt := range opts {
		opt(options)
	}

	select {
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck
	default:
		if exists, err := m.IsExist(ctx, key); err != nil {
			return err
		} else if exists && !options.Overwrite {
			return ErrDuplicateKey
		}

		m.m.Lock()
		defer m.m.Unlock()

		bytes, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("%w: marshal %+v: %s", ErrSerialize, val, err.Error())
		}
		m.data[key] = bytes
		return nil
	}
}

func FilterAll[T any](ctx context.Context, m *MStorage, fn func(T) bool) ([]T, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	var result = make([]T, 0, len(m.data))
	for _, bytes := range m.data {
		select {
		case <-ctx.Done():
			return nil, ctx.Err() //nolint:wrapcheck
		default:
			var val T
			if err := json.Unmarshal(bytes, &val); err != nil {
				return nil, fmt.Errorf("failed to unmarshal json for object `%+v`: %w", val, err)
			}
			if fn(val) {
				result = append(result, val)
			}
		}
	}
	return result, nil
}

func GetAll[T any](ctx context.Context, m *MStorage) ([]T, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	var result = make([]T, 0, len(m.data))

	for _, bytes := range m.data {
		select {
		case <-ctx.Done():
			return nil, ctx.Err() //nolint:wrapcheck
		default:
		}

		var val T
		if err := json.Unmarshal(bytes, &val); err != nil {
			return nil, fmt.Errorf("failed to unmarshal json for object `%+v`: %w", val, err)
		}
		result = append(result, val)
	}
	return result, nil
}
