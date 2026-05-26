package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

func newTestSagaStore(t *testing.T) (*SQLSagaStore, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	store, err := NewSQLSagaStore(db)
	if err != nil {
		t.Fatalf("NewSQLSagaStore: %v", err)
	}

	return store, mock
}

func TestNewSQLSagaStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSQLSagaStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteSagaStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSQLiteSagaStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestSQLSagaStore_Save_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestSagaStore(t)
	ctx := context.Background()

	state := &saga.State{
		ID:          id.NewAggregateID(),
		SagaType:    "order",
		Status:      saga.StatusRunning,
		CurrentStep: 1,
		ErrMsg:      "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO sagas").
		WithArgs(state.ID.String(), state.SagaType, string(state.Status), state.CurrentStep, state.ErrMsg, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestSQLSagaStore_Save_NilState(t *testing.T) {
	t.Parallel()

	store, _ := newTestSagaStore(t)
	ctx := context.Background()

	err := store.Save(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil state")
	}
}

func TestSQLSagaStore_Load_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestSagaStore(t)
	ctx := context.Background()

	stateID := id.NewAggregateID()
	now := time.Now()

	mock.ExpectQuery("SELECT saga_type, status, current_step, err_msg, created_at, updated_at FROM sagas").
		WithArgs(stateID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"saga_type", "status", "current_step", "err_msg", "created_at", "updated_at"}).
			AddRow("order", string(saga.StatusRunning), 2, "", now, now))

	loaded, err := store.Load(ctx, stateID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != stateID {
		t.Errorf("ID mismatch: got %v, want %v", loaded.ID, stateID)
	}
	if loaded.SagaType != "order" {
		t.Errorf("SagaType mismatch: got %q, want %q", loaded.SagaType, "order")
	}
	if loaded.Status != saga.StatusRunning {
		t.Errorf("Status mismatch: got %q, want %q", loaded.Status, saga.StatusRunning)
	}
	if loaded.CurrentStep != 2 {
		t.Errorf("CurrentStep mismatch: got %d, want %d", loaded.CurrentStep, 2)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestSQLSagaStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store, mock := newTestSagaStore(t)
	ctx := context.Background()

	stateID := id.NewAggregateID()

	mock.ExpectQuery("SELECT saga_type, status, current_step, err_msg, created_at, updated_at FROM sagas").
		WithArgs(stateID.String()).
		WillReturnError(sql.ErrNoRows)

	_, err := store.Load(ctx, stateID)
	if err == nil {
		t.Fatal("expected error for not found")
	}

	if !errors.Is(err, saga.ErrSagaNotFound) {
		t.Fatalf("expected ErrSagaNotFound, got: %v", err)
	}
}

func TestSQLSagaStore_LoadAllRunning_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestSagaStore(t)
	ctx := context.Background()

	stateID := id.NewAggregateID()
	now := time.Now()

	mock.ExpectQuery("SELECT id, saga_type, status, current_step, err_msg, created_at, updated_at FROM sagas").
		WithArgs(string(saga.StatusRunning), string(saga.StatusCompensating)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "saga_type", "status", "current_step", "err_msg", "created_at", "updated_at"}).
			AddRow(stateID.String(), "order", string(saga.StatusRunning), 1, "", now, now))

	states, err := store.LoadAllRunning(ctx)
	if err != nil {
		t.Fatalf("LoadAllRunning: %v", err)
	}

	if len(states) != 1 {
		t.Fatalf("expected 1 running saga, got %d", len(states))
	}

	if states[0].ID != stateID {
		t.Errorf("ID mismatch: got %v, want %v", states[0].ID, stateID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestSQLSagaStore_LoadAllRunning_Empty(t *testing.T) {
	t.Parallel()

	store, mock := newTestSagaStore(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id, saga_type, status, current_step, err_msg, created_at, updated_at FROM sagas").
		WithArgs(string(saga.StatusRunning), string(saga.StatusCompensating)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "saga_type", "status", "current_step", "err_msg", "created_at", "updated_at"}))

	states, err := store.LoadAllRunning(ctx)
	if err != nil {
		t.Fatalf("LoadAllRunning: %v", err)
	}

	if len(states) != 0 {
		t.Errorf("expected 0 running sagas, got %d", len(states))
	}
}

func TestSagaSchema(t *testing.T) {
	t.Parallel()

	schema := SagaSchema()
	if schema == "" {
		t.Fatal("expected non-empty schema")
	}
}

func TestSQLiteSagaSchema(t *testing.T) {
	t.Parallel()

	schema := SQLiteSagaSchema()
	if schema == "" {
		t.Fatal("expected non-empty schema")
	}
}
