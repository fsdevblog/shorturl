// Package memstore предоставляет реализацию репозитория URL для in-memory хранилища.
//
// Все методы репозитория преобразуют внутренние ошибки хранилища в общие ошибки уровня репозитория
// с помощью convertErrorType:
//   - memory.ErrDuplicateKey -> repositories.ErrDuplicateKey
//   - memory.ErrNotFound -> repositories.ErrNotFound
//   - другие ошибки -> repositories.ErrUnknown
package memstore
