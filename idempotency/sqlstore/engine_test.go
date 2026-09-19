package sqlstore_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	sqlstore "github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	idempotency "github.com/larsartmann/go-idempotency"
)

func newEngineStore(t *testing.T) *sqlstore.Store {
	t.Helper()

	eng := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = eng.Close() })

	dedup, err := metaengine.NewMapDedupStore(eng)
	if err != nil {
		t.Fatalf("NewMapDedupStore: %v", err)
	}

	store, err := sqlstore.NewFromEngine(dedup, "")
	if err != nil {
		t.Fatalf("NewFromEngine: %v", err)
	}

	return store
}

// TestEngineFacade_APIParity: the engine-backed facade honors the SQL
// store's public contract — CAS duplicate signal, TTL window, sweep — over
// any metaengine.DedupStore (here: the Map runtime on the memory engine).
func TestEngineFacade_APIParity(t *testing.T) {
	t.Parallel()

	store := newEngineStore(t)
	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "cmd-1", time.Minute); err != nil {
		t.Fatalf("first CheckAndRecord: %v", err)
	}

	if err := store.CheckAndRecord(
		ctx,
		"cmd-1",
		time.Minute,
	); !errors.Is(
		err,
		idempotency.ErrDuplicate,
	) {
		t.Fatalf("duplicate CheckAndRecord: want ErrDuplicate, got %v", err)
	}

	seen, err := store.Seen(ctx, "cmd-1")
	mustT(t, err)

	if !seen {
		t.Fatal("Seen must be true inside the window")
	}

	if err := store.Record(ctx, "cmd-2", time.Minute); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := store.Record(ctx, "cmd-2", time.Minute); err != nil {
		t.Fatalf("Record re-record must be a no-op: %v", err)
	}

	if err := store.CheckAndRecord(ctx, "no-ttl", 0); !errors.Is(err, idempotency.ErrInvalidTTL) {
		t.Fatalf("zero TTL: want ErrInvalidTTL, got %v", err)
	}
}

func TestEngineFacade_ConcurrentExactlyOneWinner(t *testing.T) {
	t.Parallel()

	store := newEngineStore(t)
	ctx := context.Background()

	const racers = 16

	var wg sync.WaitGroup

	winners := make(chan bool, racers)

	for range racers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			err := store.CheckAndRecord(ctx, "race", time.Minute)
			if err != nil && !errors.Is(err, idempotency.ErrDuplicate) {
				t.Errorf("racer: %v", err)

				return
			}

			winners <- !errors.Is(err, idempotency.ErrDuplicate)
		}()
	}

	wg.Wait()
	close(winners)

	firsts := 0

	for won := range winners {
		if won {
			firsts++
		}
	}

	if firsts != 1 {
		t.Fatalf("CAS violated: %d first-recordings, want exactly 1", firsts)
	}
}

func mustT(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
