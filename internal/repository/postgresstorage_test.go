package repository

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockDB создаёт *sql.DB с sqlmock и регистрирует Cleanup для проверки ожиданий.
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func TestNewPostgresStorage(t *testing.T) {
	db, _ := newMockDB(t)

	s := NewPostgresStorage(db, nil)
	assert.NotNil(t, s)
	assert.NotNil(t, s.logger)
}

func TestPostgresStorage_Close(t *testing.T) {
	db, _ := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	assert.NoError(t, s.Close())
}

func TestPostgresStorage_UpdateMetric_Gauge(t *testing.T) {
	db, mock := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	ctx := context.Background()

	mock.ExpectExec(`INSERT INTO metrics`).
		WithArgs("cpu", models.Gauge, 0.75).
		WillReturnResult(sqlmock.NewResult(1, 1))

	val := 0.75
	err := s.UpdateMetric(ctx, models.Metrics{ID: "cpu", MType: models.Gauge, Value: &val})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_UpdateMetric_Counter(t *testing.T) {
	db, mock := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	ctx := context.Background()

	mock.ExpectExec(`INSERT INTO metrics`).
		WithArgs("requests", models.Counter, int64(10)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	delta := int64(10)
	err := s.UpdateMetric(ctx, models.Metrics{ID: "requests", MType: models.Counter, Delta: &delta})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_UpdateMetric_NilGaugeValue(t *testing.T) {
	db, _ := newMockDB(t)
	s := NewPostgresStorage(db, nil)

	err := s.UpdateMetric(context.Background(), models.Metrics{ID: "cpu", MType: models.Gauge, Value: nil})
	assert.Error(t, err)
}

func TestPostgresStorage_UpdateMetric_NilCounterDelta(t *testing.T) {
	db, _ := newMockDB(t)
	s := NewPostgresStorage(db, nil)

	err := s.UpdateMetric(context.Background(), models.Metrics{ID: "c", MType: models.Counter, Delta: nil})
	assert.Error(t, err)
}

func TestPostgresStorage_UpdateMetric_UnknownType(t *testing.T) {
	db, _ := newMockDB(t)
	s := NewPostgresStorage(db, nil)

	err := s.UpdateMetric(context.Background(), models.Metrics{ID: "x", MType: "unknown"})
	assert.Error(t, err)
}

func TestPostgresStorage_UpdateMetrics_Empty(t *testing.T) {
	db, _ := newMockDB(t)
	s := NewPostgresStorage(db, nil)

	err := s.UpdateMetrics(context.Background(), nil)
	assert.NoError(t, err)
}

func TestPostgresStorage_UpdateMetrics_Batch(t *testing.T) {
	db, mock := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO metrics \(id, type, value\)`)
	mock.ExpectPrepare(`INSERT INTO metrics \(id, type, delta\)`)
	mock.ExpectExec(`INSERT INTO metrics \(id, type, value\)`).
		WithArgs("cpu", models.Gauge, 1.5).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO metrics \(id, type, delta\)`).
		WithArgs("req", models.Counter, int64(5)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	v := 1.5
	d := int64(5)
	err := s.UpdateMetrics(ctx, []models.Metrics{
		{ID: "cpu", MType: models.Gauge, Value: &v},
		{ID: "req", MType: models.Counter, Delta: &d},
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_GetMetric_Gauge(t *testing.T) {
	db, mock := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"value", "delta"}).
		AddRow(3.14, nil)
	mock.ExpectQuery(`SELECT value, delta FROM metrics`).
		WithArgs("pi", models.Gauge).
		WillReturnRows(rows)

	m, err := s.GetMetric(ctx, "pi", models.Gauge)
	require.NoError(t, err)
	require.NotNil(t, m.Value)
	assert.Equal(t, 3.14, *m.Value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_GetMetric_Counter(t *testing.T) {
	db, mock := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"value", "delta"}).
		AddRow(nil, int64(42))
	mock.ExpectQuery(`SELECT value, delta FROM metrics`).
		WithArgs("hits", models.Counter).
		WillReturnRows(rows)

	m, err := s.GetMetric(ctx, "hits", models.Counter)
	require.NoError(t, err)
	require.NotNil(t, m.Delta)
	assert.Equal(t, int64(42), *m.Delta)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_GetMetric_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT value, delta FROM metrics`).
		WithArgs("missing", models.Gauge).
		WillReturnError(sql.ErrNoRows)

	_, err := s.GetMetric(ctx, "missing", models.Gauge)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_GetAllMetrics(t *testing.T) {
	db, mock := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "type", "value", "delta"}).
		AddRow("cpu", models.Gauge, 0.5, nil).
		AddRow("hits", models.Counter, nil, int64(100))
	mock.ExpectQuery(`SELECT id, type, value, delta FROM metrics`).
		WillReturnRows(rows)

	all, err := s.GetAllMetrics(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_GetAllMetrics_Empty(t *testing.T) {
	db, mock := newMockDB(t)
	s := NewPostgresStorage(db, nil)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "type", "value", "delta"})
	mock.ExpectQuery(`SELECT id, type, value, delta FROM metrics`).
		WillReturnRows(rows)

	all, err := s.GetAllMetrics(ctx)
	require.NoError(t, err)
	assert.Empty(t, all)
	assert.NoError(t, mock.ExpectationsWereMet())
}
