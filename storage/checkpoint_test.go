package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

func newTestCheckpointStore(t *testing.T) (*SQLCheckpointStore, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	store, err := NewSQLCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewSQLCheckpointStore: %v", err)
	}

	return store, mock
}

func TestSQLCheckpointStore_Close(t *testing.T) {
	t.Parallel()

	s, _ := newTestCheckpointStore(t)

	err := s.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestSQLCheckpointStore_Load(t *testing.T) {
	t.Parallel()

	s, mock := newTestCheckpointStore(t)
	eventID := id.NewEventID()

	mock.ExpectQuery(`SELECT event_id, processed_at FROM checkpoints`).
		WithArgs("my-projection").
		WillReturnRows(sqlmock.NewRows([]string{"event_id", "processed_at"}).
			AddRow(eventID.String(), time.Now()))

	got, err := s.Load(context.Background(), "my-projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.EventID != eventID {
		t.Fatalf("got %v, want %v", got.EventID, eventID)
	}
}

func TestSQLCheckpointStore_Load_NoRows(t *testing.T) {
	t.Parallel()

	s, mock := newTestCheckpointStore(t)

	mock.ExpectQuery(`SELECT event_id, processed_at FROM checkpoints`).
		WithArgs("new-projection").
		WillReturnError(sql.ErrNoRows)

	got, err := s.Load(context.Background(), "new-projection")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !got.IsZero() {
		t.Fatalf("expected zero Checkpoint for missing checkpoint, got %v", got)
	}
}

func TestSQLCheckpointStore_Load_QueryError(t *testing.T) {
	t.Parallel()

	s, mock := newTestCheckpointStore(t)

	mock.ExpectQuery(`SELECT event_id, processed_at FROM checkpoints`).
		WithArgs("my-projection").
		WillReturnError(errors.New("connection lost"))

	_, err := s.Load(context.Background(), "my-projection")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSQLCheckpointStore_Load_InvalidEventID(t *testing.T) {
	t.Parallel()

	s, mock := newTestCheckpointStore(t)

	mock.ExpectQuery(`SELECT event_id, processed_at FROM checkpoints`).
		WithArgs("my-projection").
		WillReturnRows(sqlmock.NewRows([]string{"event_id", "processed_at"}).
			AddRow("not-a-valid-ulid", time.Now()))

	_, err := s.Load(context.Background(), "my-projection")
	if err == nil {
		t.Fatal("expected error for invalid event ID")
	}
}

func TestSQLCheckpointStore_Save(t *testing.T) {
	t.Parallel()

	s, mock := newTestCheckpointStore(t)
	cp := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}

	mock.ExpectExec(`INSERT INTO checkpoints`).
		WithArgs("my-projection", cp.EventID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.Save(context.Background(), "my-projection", cp)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestSQLCheckpointStore_Save_Error(t *testing.T) {
	t.Parallel()

	s, mock := newTestCheckpointStore(t)
	cp := event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()}

	mock.ExpectExec(`INSERT INTO checkpoints`).
		WithArgs("my-projection", cp.EventID, sqlmock.AnyArg()).
		WillReturnError(errors.New("db error"))

	err := s.Save(context.Background(), "my-projection", cp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckpointSchema_ContainsExpectedDDL(t *testing.T) {
	t.Parallel()

	schema := CheckpointSchema()

	for _, want := range []string{"CREATE TABLE", "checkpoints", "projection_name", "event_id", "processed_at"} {
		if !strings.Contains(schema, want) {
			t.Errorf("schema missing %q", want)
		}
	}
}
