package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/goccy/go-json"
)

// MStorage реализует in-memory хранилище данных с потокобезопасным доступом.
type MStorage struct {
	data map[string][]byte
	m    sync.RWMutex
}

// NewMemStorage создает новый экземпляр in-memory хранилища.
//
// Возвращает:
//   - *MStorage: инициализированное хранилище
func NewMemStorage() *MStorage {
	return &MStorage{
		data: make(map[string][]byte),
	}
}

// Len возвращает количество элементов в хранилище.
//
// Возвращает:
//   - int: количество элементов
func (m *MStorage) Len() int {
	return len(m.data)
}

// Ping проверяет доступность хранилища (заглушка для совместимости интерфейсов).
//
// Параметры:
//   - ctx: контекст выполнения
//
// Возвращает:
//   - error: всегда nil
func (m *MStorage) Ping(_ context.Context) error {
	return nil
}

// IsExist проверяет существование ключа в хранилище.
//
// Параметры:
//   - ctx: контекст выполнения
//   - key: проверяемый ключ
//
// Возвращает:
//   - bool: true если ключ существует
//   - error: ошибка проверки
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

// Get получает значение по ключу и десериализует его в указанный тип.
//
// Параметры:
//   - ctx: контекст выполнения
//   - key: ключ для получения значения
//   - m: хранилище
//
// Возвращает:
//   - *T: указатель на десериализованное значение
//   - error: ошибка получения или десериализации
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

// SetOptions опции для операции сохранения.
type SetOptions struct {
	Overwrite bool // разрешить перезапись существующего значения
}

// WithOverwrite создает опцию разрешающую перезапись существующего значения.
func WithOverwrite() func(*SetOptions) {
	return func(o *SetOptions) {
		o.Overwrite = true
	}
}

// BatchResult результат одной операции пакетного сохранения.
type BatchResult struct {
	Key string // Ключ
	Err error  // Ошибка операции
}

// BatchSet выполняет пакетное сохранение значений.
//
// Параметры:
//   - ctx: контекст выполнения
//   - values: карта ключ-значение для сохранения
//   - m: хранилище
//   - opts: опции сохранения
//
// Возвращает:
//   - []BatchResult: результаты операций сохранения
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

// Set сохраняет значение по ключу.
//
// Параметры:
//   - ctx: контекст выполнения
//   - key: ключ
//   - val: сохраняемое значение
//   - m: хранилище
//   - opts: опции сохранения
//
// Возвращает:
//   - error: ошибка сохранения
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

// FilterAll возвращает все значения, удовлетворяющие предикату.
//
// Параметры:
//   - ctx: контекст выполнения
//   - m: хранилище
//   - fn: функция-предикат
//
// Возвращает:
//   - []T: отфильтрованные значения
//   - error: ошибка фильтрации
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

// GetAll возвращает все значения из хранилища.
//
// Параметры:
//   - ctx: контекст выполнения
//   - m: хранилище
//
// Возвращает:
//   - []T: все значения
//   - error: ошибка получения
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
