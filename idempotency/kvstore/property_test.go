package kvstore_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4"
	idemsqlstore "github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

var propertyDBCounter atomic.Int64

// storeFactory creates a fresh, isolated idempotency.Store for property testing.
// Each call returns a new store and a cleanup function.
type storeFactory func(t *testing.T) (idempotency.Store, func())

// allStores returns the 3 production Store implementations as a table for
// cross-implementation property testing. Each factory creates a fresh instance
// so rapid iterations never carry state from a previous draw.
func allStores() map[string]storeFactory {
	return map[string]storeFactory{
		"memory": func(t *testing.T) (idempotency.Store, func()) {
			s := idempotency.NewMemoryStore(0)

			return s, func() { s.Close() }
		},
		"kvstore": func(t *testing.T) (idempotency.Store, func()) {
			s := kvstore.New(kv.NewMemStore())

			return s, func() { _ = s.Close() }
		},
		"sqlstore": func(t *testing.T) (idempotency.Store, func()) {
			return newSQLiteStoreForProperty(t), func() {}
		},
	}
}

// newSQLiteStoreForProperty creates a SQLite store backed by a UNIQUE named
// in-memory database, so parallel property-test iterations never share state.
func newSQLiteStoreForProperty(t *testing.T) idempotency.Store {
	t.Helper()
	dbName := fmt.Sprintf("propertydb_%d", propertyDBCounter.Add(1))
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=busy_timeout(5000)", dbName)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	s, err := idemsqlstore.NewSQLiteStore(context.Background(), db)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}

	return s
}

// runPropertyAllStores runs a rapid property check against every implementation.
func runPropertyAllStores(
	t *testing.T,
	fn func(t *rapid.T, store idempotency.Store),
) {
	t.Helper()
	for name, factory := range allStores() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rapid.Check(t, func(rt *rapid.T) {
				store, cleanup := factory(t)
				defer cleanup()
				fn(rt, store)
			})
		})
	}
}

// TestProperty_RecordIsIdempotent_AllStores: Recording the same key multiple
// times never errors and the key remains seen, across all implementations.
func TestProperty_RecordIsIdempotent_AllStores(t *testing.T) {
	t.Parallel()

	runPropertyAllStores(t, func(rt *rapid.T, store idempotency.Store) {
		key := rapid.String().Draw(rt, "key")
		ttl := time.Duration(rapid.IntRange(2, 60).Draw(rt, "ttl_seconds")) * time.Second

		for range rapid.IntRange(2, 10).Draw(rt, "repeats") {
			if err := store.Record(context.Background(), key, ttl); err != nil {
				rt.Fatalf("Record attempt: %v", err)
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

// TestProperty_CheckAndRecordExactlyOnce_AllStores: Among N concurrent
// CheckAndRecord calls for the same key, exactly one succeeds and the rest get
// ErrDuplicate, across all implementations.
func TestProperty_CheckAndRecordExactlyOnce_AllStores(t *testing.T) {
	t.Parallel()

	runPropertyAllStores(t, func(rt *rapid.T, store idempotency.Store) {
		key := rapid.String().Draw(rt, "key")
		n := rapid.IntRange(2, 15).Draw(rt, "concurrent")
		ttl := time.Minute

		results := make(chan error, n)
		var wg sync.WaitGroup
		wg.Add(n)
		start := make(chan struct{})
		for range n {
			go func() {
				defer wg.Done()
				<-start
				results <- store.CheckAndRecord(context.Background(), key, ttl)
			}()
		}
		close(start)
		wg.Wait()

		var wins, dups atomic.Int64
		for range n {
			err := <-results
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, idempotency.ErrDuplicate):
				dups.Add(1)
			default:
				rt.Fatalf("unexpected error: %v", err)
			}
		}

		if wins.Load() != 1 {
			rt.Fatalf("expected exactly 1 success, got %d (of %d)", wins.Load(), n)
		}
		if dups.Load() != int64(n-1) {
			rt.Fatalf("expected %d duplicates, got %d", n-1, dups.Load())
		}
	})
}

// TestProperty_KeysAreIndependent_AllStores: Operations on one key don't
// affect another, across all implementations.
func TestProperty_KeysAreIndependent_AllStores(t *testing.T) {
	t.Parallel()

	runPropertyAllStores(t, func(rt *rapid.T, store idempotency.Store) {
		keyA := rapid.StringMatching(`.+`).Draw(rt, "keyA")
		keyB := rapid.StringMatching(`.+`).
			Filter(func(s string) bool { return s != keyA }).
			Draw(rt, "keyB")
		ttl := time.Minute

		if err := store.Record(context.Background(), keyA, ttl); err != nil {
			rt.Fatalf("Record A: %v", err)
		}

		seenB, err := store.Seen(context.Background(), keyB)
		if err != nil {
			rt.Fatalf("Seen B: %v", err)
		}
		if seenB {
			rt.Fatal("keyB should not be seen (only keyA was recorded)")
		}

		if err := store.CheckAndRecord(context.Background(), keyB, ttl); err != nil {
			rt.Fatalf("CheckAndRecord B: %v", err)
		}

		seenA, err := store.Seen(context.Background(), keyA)
		if err != nil {
			rt.Fatalf("Seen A: %v", err)
		}
		if !seenA {
			rt.Fatal("keyA should still be seen after operating on keyB")
		}
	})
}

// TestProperty_TTLExpiry_AllStores: After TTL expires, Seen returns false and
// CheckAndRecord succeeds again, across all implementations.
func TestProperty_TTLExpiry_AllStores(t *testing.T) {
	t.Parallel()

	runPropertyAllStores(t, func(rt *rapid.T, store idempotency.Store) {
		key := rapid.String().Draw(rt, "key")
		ttl := 50 * time.Millisecond

		if err := store.CheckAndRecord(context.Background(), key, ttl); err != nil {
			rt.Fatalf("CheckAndRecord (first): %v", err)
		}

		time.Sleep(ttl + 30*time.Millisecond)

		seen, err := store.Seen(context.Background(), key)
		if err != nil {
			rt.Fatalf("Seen after TTL: %v", err)
		}
		if seen {
			rt.Fatal("key should not be seen after TTL expiry")
		}

		if err := store.CheckAndRecord(context.Background(), key, ttl); err != nil {
			rt.Fatalf("CheckAndRecord after TTL expiry should succeed: %v", err)
		}
	})
}
