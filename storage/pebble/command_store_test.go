package pebble_test

import (
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/command/v4/commandtest"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

func newCommandStore(t *testing.T) *cqrspebble.CommandStore {
	t.Helper()

	dir := t.TempDir()

	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewCommandStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func TestCommandStoreSuite(t *testing.T) {
	t.Parallel()

	commandtest.RunStoreSuite(t, func(t *testing.T) commandtest.StoreSuite {
		return newCommandStore(t)
	})
}
