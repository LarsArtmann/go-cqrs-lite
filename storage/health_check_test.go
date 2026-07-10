package storage

import (
	"context"
	"testing"
)

func TestSQLEventStore_HealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	if err := store.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck on healthy store: %v", err)
	}
}

func TestSQLEventStore_HealthCheck_Closed(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	_ = store.Close()

	if err := store.HealthCheck(context.Background()); err == nil {
		t.Error("HealthCheck on closed store should return error")
	}
}
