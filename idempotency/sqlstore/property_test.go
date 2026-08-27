package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	_ "modernc.org/sqlite"
	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
)

var propDBCounter atomic.Int64

// ttlTestParams returns a (ttl, wait) pair with enough headroom to survive
// scheduler jitter under -race without making the non-race path slow. Shared
// by the TTL property test (and the build-tagged integration tests).
//
// The non-race margins were widened from 10ms/50ms after a full workspace
// gate run (all modules in parallel, high ambient load) descheduled the test
// goroutine for longer than the 10ms TTL between Record and the immediate
// Seen check, legitimately expiring the key before first observation.
// 50ms/250ms covers realistic gate-load jitter while keeping the test fast.
func ttlTestParams() (time.Duration, time.Duration) {
	if raceEnabled {
		return 100 * time.Millisecond, 400 * time.Millisecond
	}

	return 50 * time.Millisecond, 250 * time.Millisecond
}

func newTestStore(tb testing.TB) (*sqlstore.Store, *sql.DB) {
	tb.Helper()

	// Each test gets its own in-memory database via a unique name.
	// Using file::memory:?cache=shared with the SAME name causes cross-test
	// interference. A unique DSN per call avoids this.
	n := propDBCounter.Add(1)
	dsn := fmt.Sprintf("file:propdb%d?mode=memory&cache=shared&_pragma=busy_timeout(5000)", n)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		tb.Fatalf("open sqlite: %v", err)
	}

	store, err := sqlstore.NewSQLiteStore(context.Background(), db)
	if err != nil {
		tb.Fatalf("create store: %v", err)
	}

	tb.Cleanup(func() {
		store.Close()
		db.Close()
	})

	return store, db
}

// TestProperty_SQLiteRecordIsIdempotent: Recording the same key multiple times
// never errors and the key remains seen.
func TestProperty_SQLiteRecordIsIdempotent(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store, _ := newTestStore(t)

		key := rapid.String().Draw(rt, "key")
		ttl := time.Duration(rapid.IntRange(1, 60).Draw(rt, "ttl_seconds")) * time.Second

		for i := 0; i < rapid.IntRange(2, 10).Draw(rt, "repeats"); i++ {
			if err := store.Record(context.Background(), key, ttl); err != nil {
				rt.Fatalf("Record attempt %d: %v", i, err)
			}
		}

		seen, err := store.Seen(context.Background(), key)
		if err != nil {
			rt.Fatalf("Seen: %v", err)
		}

		if !seen {
			rt.Fatal("key should be seen after repeated Record calls")
		}
	})
}

// TestProperty_SQLiteCheckAndRecordExactlyOnce: Among N concurrent
// CheckAndRecord calls for the same key, exactly one succeeds and the rest
// get ErrDuplicate.
func TestProperty_SQLiteCheckAndRecordExactlyOnce(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store, _ := newTestStore(t)

		key := rapid.String().Draw(rt, "key")
		n := rapid.IntRange(2, 10).Draw(rt, "concurrent")
		ttl := time.Minute

		results := make(chan error, n)

		var wg sync.WaitGroup

		for i := 0; i < n; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				results <- store.CheckAndRecord(context.Background(), key, ttl)
			}()
		}

		wg.Wait()
		close(results)

		successes, duplicates := 0, 0

		for err := range results {
			if err == nil {
				successes++
			} else if errors.Is(err, idempotency.ErrDuplicate) {
				duplicates++
			} else {
				rt.Fatalf("unexpected error: %v", err)
			}
		}

		if successes != 1 {
			rt.Fatalf("expected exactly 1 success, got %d", successes)
		}

		if duplicates != n-1 {
			rt.Fatalf("expected %d duplicates, got %d", n-1, duplicates)
		}
	})
}

// TestProperty_SQLiteTTLExpiry: A key recorded with TTL T becomes unseen
// after T elapses.
func TestProperty_SQLiteTTLExpiry(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Create and close the store per-iteration to avoid accumulating
		// hundreds of open SQLite connections (newTestStore registers
		// cleanup on t, not rt, so connections persist until test end).
		n := propDBCounter.Add(1)
		dsn := fmt.Sprintf("file:propdb%d?mode=memory&cache=shared&_pragma=busy_timeout(5000)", n)
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			rt.Fatalf("open sqlite: %v", err)
		}
		defer db.Close()

		store, err := sqlstore.NewSQLiteStore(context.Background(), db)
		if err != nil {
			rt.Fatalf("create store: %v", err)
		}
		defer store.Close()

		key := rapid.String().Draw(rt, "key")
		// Suffix guarantees visKey != key: equal drawn strings would make the
		// 1-hour visibility row shadow the TTL row (Record is ON CONFLICT DO
		// NOTHING, so the second write would be silently dropped).
		visKey := key + "|cqrs-vis|"
		// Expiry margins: see ttlTestParams. The visibility assertion uses a
		// LONG TTL so it cannot be defeated by scheduler jitter descheduling
		// the goroutine past the TTL between Record and Seen (the flake class
		// that margin bumps alone never fully killed).
		ttl, wait := ttlTestParams()

		if err := store.Record(context.Background(), visKey, time.Hour); err != nil {
			rt.Fatalf("Record visKey: %v", err)
		}

		seen, err := store.Seen(context.Background(), visKey)
		if err != nil {
			rt.Fatalf("Seen visKey: %v", err)
		}

		if !seen {
			rt.Fatal("key should be seen immediately after Record")
		}

		if err := store.Record(context.Background(), key, ttl); err != nil {
			rt.Fatalf("Record: %v", err)
		}

		time.Sleep(wait)

		seen, err = store.Seen(context.Background(), key)
		if err != nil {
			rt.Fatalf("Seen after expiry: %v", err)
		}

		if seen {
			rt.Fatal("key should be unseen after TTL expiry")
		}
	})
}
