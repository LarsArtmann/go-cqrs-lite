package readmodel

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// TestSQLKVStore_CRUDRoundtrip is a smoke test that exercises the readmodel
// package directly (not through the storage aliases), proving the sub-package
// is independently usable and has local test coverage.
func TestSQLKVStore_CRUDRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(ctx, sqlpkg.SQLiteSchemaEmbed()); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	store, err := NewSQLiteKVStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteKVStore: %v", err)
	}

	if _, err := store.Get(ctx, []byte("missing")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("Get missing: err = %v, want ErrNotFound", err)
	}

	if err := store.Set(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(got) != "v1" {
		t.Fatalf("Get: got %q, want %q", got, "v1")
	}
}

func TestNewSQLiteKVStore_NilDB(t *testing.T) {
	t.Parallel()

	if _, err := NewSQLiteKVStore(nil); err == nil {
		t.Fatal("expected error for nil db")
	}
}
