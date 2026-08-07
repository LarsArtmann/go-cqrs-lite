package pebble_test

import (
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/query/v4/querytest"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

func newQueryStore(t *testing.T) *cqrspebble.QueryStore {
	t.Helper()

	dir := t.TempDir()

	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	store, err := cqrspebble.NewQueryStore(database, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func TestQueryStoreSuite(t *testing.T) {
	t.Parallel()

	querytest.RunStoreSuite(t, func(t *testing.T) querytest.StoreSuite {
		return newQueryStore(t)
	})
}
