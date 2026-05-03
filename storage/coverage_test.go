package storage_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage"
)

func TestNewSQLEventStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := storage.NewSQLEventStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLSnapshotStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := storage.NewSQLSnapshotStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewSQLCheckpointStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := storage.NewSQLCheckpointStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}
