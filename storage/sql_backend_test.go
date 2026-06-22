package storage

import (
	"testing"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

func newTestSQLBackend(t *testing.T) *SQLBackend {
	t.Helper()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	backend, err := NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}

	return backend
}

func TestNewSQLBackend_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSQLBackend(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLiteBackend_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSQLiteBackend(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestSQLBackend_EventStore(t *testing.T) {
	t.Parallel()

	backend := newTestSQLBackend(t)

	store := backend.EventStore()
	if store == nil {
		t.Fatal("expected non-nil EventStore")
	}
}

func TestNewSQLBackendWithDialect(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	backend, err := NewSQLBackendWithDialect(db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewSQLBackendWithDialect: %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
	if backend.EventStore() == nil {
		t.Fatal("expected non-nil EventStore")
	}
}
