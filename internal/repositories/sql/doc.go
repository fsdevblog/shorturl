// Package sql предоставляет реализацию репозитория URL для PostgreSQL.
//
// Все методы репозитория преобразуют ошибки PostgreSQL в общие ошибки уровня репозитория
// с помощью convertErrType:
//   - uniqueViolationCode (23505) -> repositories.ErrDuplicateKey
//   - другие ошибки -> repositories.ErrUnknown
package sql
