package storage

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestOutboxStatus_String(t *testing.T) {
	t.Parallel()

	if OutboxStatusPending.String() != "pending" {
		t.Errorf("OutboxStatusPending.String() = %q, want %q", OutboxStatusPending.String(), "pending")
	}

	if OutboxStatusAcked.String() != "acked" {
		t.Errorf("OutboxStatusAcked.String() = %q, want %q", OutboxStatusAcked.String(), "acked")
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
