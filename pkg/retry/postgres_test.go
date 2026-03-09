package retry

import (
	"database/sql"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// pgErr конструирует pgconn.PgError с заданным кодом ошибки.
func pgErr(code string) *pgconn.PgError {
	return &pgconn.PgError{Code: code}
}

func TestNewPostgresErrorClassifier(t *testing.T) {
	c := NewPostgresErrorClassifier()
	assert.NotNil(t, c)
}

func TestClassify_Nil(t *testing.T) {
	c := NewPostgresErrorClassifier()
	assert.Equal(t, NonRetriable, c.Classify(nil))
}

func TestClassify_ErrConnDone(t *testing.T) {
	c := NewPostgresErrorClassifier()
	assert.Equal(t, Retriable, c.Classify(sql.ErrConnDone))
}

func TestClassify_ConnectionException(t *testing.T) {
	c := NewPostgresErrorClassifier()
	// 08006 — connection failure (класс 08)
	assert.Equal(t, Retriable, c.Classify(pgErr(pgerrcode.ConnectionFailure)))
}

func TestClassify_TransactionRollback(t *testing.T) {
	c := NewPostgresErrorClassifier()
	// 40001 — serialization failure
	assert.Equal(t, Retriable, c.Classify(pgErr(pgerrcode.SerializationFailure)))
	// 40P01 — deadlock detected
	assert.Equal(t, Retriable, c.Classify(pgErr(pgerrcode.DeadlockDetected)))
}

func TestClassify_CannotConnectNow(t *testing.T) {
	c := NewPostgresErrorClassifier()
	// 57P03 — cannot connect now
	assert.Equal(t, Retriable, c.Classify(pgErr(pgerrcode.CannotConnectNow)))
}

func TestClassify_UniqueViolation(t *testing.T) {
	c := NewPostgresErrorClassifier()
	assert.Equal(t, NonRetriable, c.Classify(pgErr(pgerrcode.UniqueViolation)))
}

func TestClassify_SyntaxError(t *testing.T) {
	c := NewPostgresErrorClassifier()
	assert.Equal(t, NonRetriable, c.Classify(pgErr(pgerrcode.SyntaxError)))
}

func TestClassify_DataException(t *testing.T) {
	c := NewPostgresErrorClassifier()
	// 22001 — string data right truncation (класс 22)
	assert.Equal(t, NonRetriable, c.Classify(pgErr(pgerrcode.StringDataRightTruncationDataException)))
}

func TestClassify_UndefinedTable(t *testing.T) {
	c := NewPostgresErrorClassifier()
	assert.Equal(t, NonRetriable, c.Classify(pgErr(pgerrcode.UndefinedTable)))
}

func TestClassify_UnknownCode(t *testing.T) {
	c := NewPostgresErrorClassifier()
	// Неизвестный код — по умолчанию NonRetriable
	assert.Equal(t, NonRetriable, c.Classify(pgErr("99999")))
}

func TestIsPostgresRetriableError_True(t *testing.T) {
	assert.True(t, IsPostgresRetriableError(pgErr(pgerrcode.ConnectionFailure)))
}

func TestIsPostgresRetriableError_False(t *testing.T) {
	assert.False(t, IsPostgresRetriableError(pgErr(pgerrcode.UniqueViolation)))
	assert.False(t, IsPostgresRetriableError(nil))
}

func TestIsUniqueViolation_True(t *testing.T) {
	assert.True(t, IsUniqueViolation(pgErr(pgerrcode.UniqueViolation)))
}

func TestIsUniqueViolation_False(t *testing.T) {
	assert.False(t, IsUniqueViolation(nil))
	assert.False(t, IsUniqueViolation(pgErr(pgerrcode.ConnectionFailure)))
}

func TestGetPostgresErrorCode_WithPgError(t *testing.T) {
	code := GetPostgresErrorCode(pgErr(pgerrcode.UniqueViolation))
	assert.Equal(t, pgerrcode.UniqueViolation, code)
}

func TestGetPostgresErrorCode_Nil(t *testing.T) {
	assert.Equal(t, "", GetPostgresErrorCode(nil))
}

func TestGetPostgresErrorCode_NonPgError(t *testing.T) {
	assert.Equal(t, "", GetPostgresErrorCode(sql.ErrNoRows))
}
