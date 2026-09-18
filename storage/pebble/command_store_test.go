package pebble_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4/commandtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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

func TestCommandStore_ConcurrentSaveSameID_ExactlyOneWins(t *testing.T) {
	t.Parallel()

	store := newCommandStore(t)

	ref := command.StreamRef{Type: "Test", ID: id.NewStreamID()}

	cmd, err := command.NewPersistedCommand("user.create", ref, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	var wg sync.WaitGroup

	results := make([]error, 8)
	for i := range 8 {
		wg.Add(1)

		go func(slot int) {
			defer wg.Done()
			results[slot] = store.Save(context.Background(), ref, cmd)
		}(i)
	}

	wg.Wait()

	conflicts, others := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
		case errors.Is(err, command.ErrDuplicateCommand):
			conflicts++
		default:
			others++
		}
	}

	if conflicts != 7 || others != 0 {
		t.Fatalf(
			"expected 1 success + 7 conflicts + 0 other, got %d conflicts, %d other: %v",
			conflicts,
			others,
			results,
		)
	}
}
