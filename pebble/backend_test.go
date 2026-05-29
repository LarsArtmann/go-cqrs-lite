package pebble_test

import (
	"testing"

	pebbledb "github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/store"
	"github.com/larsartmann/go-cqrs-lite/pebble"
)

func newPebbleTestBackend(t *testing.T) store.Backend {
	t.Helper()
	dir := t.TempDir()
	db, err := pebbledb.Open(dir, &pebbledb.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return pebble.NewBackend(db)
}

func TestPebbleBackend(t *testing.T) {
	t.Parallel()
	store.RunBackendTests(t, newPebbleTestBackend)
}
