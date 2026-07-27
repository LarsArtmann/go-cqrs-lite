package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
	_ "modernc.org/sqlite"
)

func newTestStore(tb testing.TB) (*sqlstore.Store, *sql.DB) {
	tb.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	store, err := sqlstore.NewSQLiteStore(context.Background(), db)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
		db.Close()
	})

	return store, db
}

// TestProperty_SQLiteRecordIsIdempotent: Recording the same key multiple times
// never errors and the key remains seen.
func TestProperty_SQLiteRecordIsIdempotent(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		store, _ := newTestStore(t)

		key := rapid.String().Draw(t, "key")
		ttl := time.Duration(rapid.IntRange(1, 60).Draw(t, "ttl_seconds")) * time.Second

		for i := 0; i < rapid.IntRange(2, 10).Draw(t, "repeats"); i++ {
			if err := store.Record(context.Background(), key, ttl); err != nil {
				t.Fatalf("Record attempt %d: %v", i, err)
			}
		}

		seen, err := store.Seen(context.Background(), key)
		if err != nil {
			t.Fatalf("Seen: %v", err)
		}

		if !seen {
			t.Fatal("key should be seen after repeated Record calls")
		}
	})
}

// TestProperty_SQLiteCheckAndRecordExactlyOnce: Among N concurrent
// CheckAndRecord calls for the same key, exactly one succeeds and the rest
// get ErrDuplicate.
func TestProperty_SQLiteCheckAndRecordExactlyOnce(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		store, _ := newTestStore(t)

		key := rapid.String().Draw(t, "key")
		n := rapid.IntRange(2, 10).Draw(t, "concurrent")
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
				t.Fatalf("unexpected error: %v", err)
			}
		}

		if successes != 1 {
			t.Fatalf("expected exactly 1 success, got %d", successes)
		}

		if duplicates != n-1 {
			t.Fatalf("expected %d duplicates, got %d", n-1, duplicates)
		}
	})
}

// TestProperty_SQLiteTTLExpiry: A key recorded with TTL T becomes unseen
// after T elapses.
func TestProperty_SQLiteTTLExpiry(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		store, _ := newTestStore(t)

		key := rapid.String().Draw(t, "key")
		ttl := 50 * time.Millisecond

		if err := store.Record(context.Background(), key, ttl); err != nil {
			t.Fatalf("Record: %v", err)
		}

		seen, _ := store.Seen(context.Background(), key)
		if !seen {
			t.Fatal("key should be seen immediately after Record")
		}

		time.Sleep(ttl + 50*time.Millisecond)

		seen, _ = store.Seen(context.Background(), key)
		if seen {
			t.Fatal("key should be unseen after TTL expiry")
		}
	})
}
