package storage_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func newBenchViewStore(b *testing.B) (*storage.SQLViewStore[testView, testKey], context.Context) {
	b.Helper()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		b.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	store, err := storage.NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		b.Fatalf("NewSQLiteViewStore: %v", err)
	}

	return store, context.Background()
}
