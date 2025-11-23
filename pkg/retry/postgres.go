package retry

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// PGErrorClassification тип для классификации ошибок
type PGErrorClassification int

const (
	// NonRetriable - операцию не следует повторять
	NonRetriable PGErrorClassification = iota

	// Retriable - операцию можно повторить
	Retriable
)

// PostgresErrorClassifier классификатор ошибок PostgreSQL
type PostgresErrorClassifier struct{}

// NewPostgresErrorClassifier создает новый классификатор ошибок PostgreSQL
func NewPostgresErrorClassifier() *PostgresErrorClassifier {
	return &PostgresErrorClassifier{}
}

// Classify классифицирует ошибку и возвращает PGErrorClassification
func (c *PostgresErrorClassifier) Classify(err error) PGErrorClassification {
	if err == nil {
		return NonRetriable
	}

	// Проверяем стандартные ошибки sql
	if errors.Is(err, sql.ErrConnDone) {
		return Retriable
	}

	// Проверяем и конвертируем в pgconn.PgError, если это возможно
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return c.classifyPgError(pgErr)
	}

	// По умолчанию считаем ошибку неповторяемой
	return NonRetriable
}

// classifyPgError классифицирует ошибку PostgreSQL по коду
// Использует функции из pgerrcode для определения типа ошибки
func (c *PostgresErrorClassifier) classifyPgError(pgErr *pgconn.PgError) PGErrorClassification {
	// Коды ошибок PostgreSQL: https://www.postgresql.org/docs/current/errcodes-appendix.html

	// Класс 08 - Ошибки соединения (retriable)
	if pgerrcode.IsConnectionException(pgErr.Code) {
		return Retriable
	}

	// Класс 40 - Откат транзакции (retriable)
	if pgerrcode.IsTransactionRollback(pgErr.Code) {
		return Retriable
	}

	// Конкретные коды отката транзакции
	switch pgErr.Code {
	case pgerrcode.SerializationFailure, // 40001
		pgerrcode.DeadlockDetected: // 40P01
		return Retriable
	}

	// Конкретные коды ошибок соединения
	switch pgErr.Code {
	case pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.CannotConnectNow: // 57P03
		return Retriable
	}

	// Класс 23 - Нарушение ограничений целостности (non-retriable)
	// Используем pgerrcode.UniqueViolation для проверки типа ошибки
	if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
		// Проверяем конкретные типы нарушений
		if pgErr.Code == pgerrcode.UniqueViolation {
			// Нарушение уникального ограничения - non-retriable
			return NonRetriable
		}
		return NonRetriable
	}

	// Класс 22 - Ошибки данных (non-retriable)
	if pgerrcode.IsDataException(pgErr.Code) {
		return NonRetriable
	}

	// Класс 42 - Синтаксические ошибки (non-retriable)
	// Проверяем конкретные коды синтаксических ошибок
	switch pgErr.Code {
	case pgerrcode.SyntaxErrorOrAccessRuleViolation,
		pgerrcode.SyntaxError,
		pgerrcode.UndefinedColumn,
		pgerrcode.UndefinedTable,
		pgerrcode.UndefinedFunction:
		return NonRetriable
	}

	// По умолчанию считаем ошибку неповторяемой
	return NonRetriable
}

// IsPostgresRetriableError определяет, является ли ошибка PostgreSQL retriable
// Использует PostgresErrorClassifier для классификации
func IsPostgresRetriableError(err error) bool {
	classifier := NewPostgresErrorClassifier()
	return classifier.Classify(err) == Retriable
}

// IsUniqueViolation проверяет, является ли ошибка нарушением уникального ограничения
// Использует pgerrcode.UniqueViolation для определения типа ошибки
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UniqueViolation
	}

	return false
}

// GetPostgresErrorCode возвращает код ошибки PostgreSQL, если ошибка является PgError
// Возвращает пустую строку, если ошибка не является PgError
func GetPostgresErrorCode(err error) string {
	if err == nil {
		return ""
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}

	return ""
}
