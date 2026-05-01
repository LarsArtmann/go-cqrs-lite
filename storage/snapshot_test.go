package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newTestSnapshotStore(t *testing.T) (*SQLSnapshotStore, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	return NewSQLSnapshotStore(db), mock
}

func TestSQLSnapshotStore_Close(t *testing.T) {
	t.Parallel()

	s, _ := newTestSnapshotStore(t)

	err := s.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestSQLSnapshotStore_Save(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)

	snap := event.Snapshot{
		AggregateID:   id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2"),
		AggregateType: "User",
		Version:       event.Version(5),
		State:         []byte(`{"name":"John"}`),
		CreatedAt:     time.Now().UTC().Truncate(time.Millisecond),
	}

	mock.ExpectExec(`INSERT INTO snapshots`).
		WithArgs("User", snap.AggregateID, 5, snap.State, snap.CreatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.Save(context.Background(), snap)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLSnapshotStore_Save_Error(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)

	snap := event.Snapshot{
		AggregateID:   id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2"),
		AggregateType: "User",
		Version:       event.Version(1),
		State:         []byte(`{}`),
		CreatedAt:     time.Now(),
	}

	mock.ExpectExec(`INSERT INTO snapshots`).
		WillReturnError(errors.New("db error"))

	err := s.Save(context.Background(), snap)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSQLSnapshotStore_Load(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)
	aggID := id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2")
	createdAt := time.Now().UTC().Truncate(time.Millisecond)

	mock.ExpectQuery(`SELECT version, state, created_at FROM snapshots`).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows([]string{"version", "state", "created_at"}).
			AddRow(3, []byte(`{"name":"John"}`), createdAt))

	snap, err := s.Load(context.Background(), "User", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if snap.Version.Int() != 3 {
		t.Fatalf("version = %d, want 3", snap.Version)
	}

	if snap.AggregateType != "User" {
		t.Fatalf("type = %q, want %q", snap.AggregateType, "User")
	}

	if snap.AggregateID != aggID {
		t.Fatalf("ID mismatch")
	}
}

func TestSQLSnapshotStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)
	aggID := id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2")

	mock.ExpectQuery(`SELECT version, state, created_at FROM snapshots`).
		WithArgs("User", aggID).
		WillReturnError(sql.ErrNoRows)

	_, err := s.Load(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestSQLSnapshotStore_Load_QueryError(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)
	aggID := id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2")

	mock.ExpectQuery(`SELECT version, state, created_at FROM snapshots`).
		WithArgs("User", aggID).
		WillReturnError(errors.New("connection lost"))

	_, err := s.Load(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSQLSnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)
	aggID := id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2")
	createdAt := time.Now().UTC().Truncate(time.Millisecond)

	mock.ExpectQuery(`SELECT version, state, created_at FROM snapshots`).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows([]string{"version", "state", "created_at"}).
			AddRow(3, []byte(`{"name":"Jane"}`), createdAt))

	snap, err := s.LoadAtVersion(context.Background(), "User", aggID, event.Version(5))
	if err != nil {
		t.Fatalf("LoadAtVersion: %v", err)
	}

	if snap.Version.Int() != 3 {
		t.Fatalf("version = %d, want 3", snap.Version)
	}
}

func TestSQLSnapshotStore_LoadAtVersion_NotFound(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)
	aggID := id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2")

	mock.ExpectQuery(`SELECT version, state, created_at FROM snapshots`).
		WithArgs("User", aggID).
		WillReturnError(sql.ErrNoRows)

	_, err := s.LoadAtVersion(context.Background(), "User", aggID, event.Version(99))
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestSQLSnapshotStore_LoadAtVersion_VersionExceedsRequested(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)
	aggID := id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2")

	mock.ExpectQuery(`SELECT version, state, created_at FROM snapshots`).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows([]string{"version", "state", "created_at"}).
			AddRow(10, []byte(`{"name":"Jane"}`), time.Now()))

	_, err := s.LoadAtVersion(context.Background(), "User", aggID, event.Version(5))
	if err == nil {
		t.Fatal("expected error when stored version exceeds requested")
	}
}

func TestSQLSnapshotStore_Delete(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)
	aggID := id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2")

	mock.ExpectExec(`DELETE FROM snapshots`).
		WithArgs("User", aggID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.Delete(context.Background(), "User", aggID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestSQLSnapshotStore_Delete_Error(t *testing.T) {
	t.Parallel()

	s, mock := newTestSnapshotStore(t)
	aggID := id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2")

	mock.ExpectExec(`DELETE FROM snapshots`).
		WithArgs("User", aggID).
		WillReturnError(errors.New("db error"))

	err := s.Delete(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSnapshotSchema_ContainsExpectedDDL(t *testing.T) {
	t.Parallel()

	schema := SnapshotSchema()

	for _, want := range []string{"CREATE TABLE", "snapshots", "aggregate_type", "aggregate_id", "version", "state"} {
		if !strings.Contains(schema, want) {
			t.Errorf("schema missing %q", want)
		}
	}
}
