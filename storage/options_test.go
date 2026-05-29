package storage

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

func TestOutboxStatus_String(t *testing.T) {
	t.Parallel()

	if got := sqlpkg.OutboxStatusPending.String(); got != "pending" {
		t.Errorf("sqlpkg.OutboxStatusPending.String() = %q, want %q", got, "pending")
	}

	if got := sqlpkg.OutboxStatusAcked.String(); got != "acked" {
		t.Errorf("sqlpkg.OutboxStatusAcked.String() = %q, want %q", got, "acked")
	}
}

func TestNewSQLEventStoreWithOptions_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSQLEventStoreWithOptions(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLEventStoreWithOptions_WithOwnership(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	store, err := NewSQLEventStoreWithOptions(db, WithOwnership())
	if err != nil {
		t.Fatalf("NewSQLEventStoreWithOptions: %v", err)
	}

	mock.ExpectClose()

	err = store.Close()
	if err != nil {
		t.Fatalf("Close with ownership: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLEventStore_Close_WithoutOwnership(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	store, err := NewSQLEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("Close without ownership should be nil: %v", err)
	}
}
