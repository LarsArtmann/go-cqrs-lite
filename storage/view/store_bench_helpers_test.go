package view

import (
	"context"
	"testing"
)

func newBenchViewStore(b *testing.B) (*SQLViewStore[testView, testKey], context.Context) {
	b.Helper()

	db, err := openSQLiteInMemory()
	if err != nil {
		b.Fatalf("OpenSQLiteInMemory: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLiteViewStore[testView, testKey](db, testMapper())
	if err != nil {
		b.Fatalf("NewSQLiteViewStore: %v", err)
	}

	return store, context.Background()
}
