package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4"
	idemsqlstore "github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

// kvstoreTTLParams returns a (ttl, wait) pair with enough headroom to survive
// -race detector scheduling inflation (mirrors idempotency/kvstore's unit
// test params; race awareness comes from testutil here).
func kvstoreTTLParams() (time.Duration, time.Duration) {
	if testutil.RaceEnabled {
		return 100 * time.Millisecond, 400 * time.Millisecond
	}

	return 10 * time.Millisecond, 50 * time.Millisecond
}

// newSQLiteStoreForContract builds a SQL-backed idempotency store for the
// cross-implementation contract test. Shared-cache in-memory SQLite + a single
// connection so concurrent writers queue instead of erroring.
func newSQLiteStoreForContract(t *testing.T) *idemsqlstore.Store {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=busy_timeout(5000)")
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

// TestKVStore_Record_MatchesMemoryStoreContract verifies the kvstore Record/Seen
// contract matches the reference MemoryStore implementation AND the SQL store.
// All three implementations must be a no-op on an existing non-expired key and
// never extend the TTL. This is the authoritative cross-implementation contract
// test; it lives in integration/ so idempotency/kvstore does not need a SQLite
// dependency for it.
//
// Note: MemoryStore.Record intentionally re-records EXPIRED entries (sets a
// fresh TTL). This test only covers the non-expired case, which all three
// implementations agree on: Record is a no-op when the key is still valid.
func TestKVStore_Record_MatchesMemoryStoreContract(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	stores := map[string]idempotency.Store{
		"memory":   idempotency.NewMemoryStore(0),
		"kvstore":  kvstore.New(kv.NewMemStore()),
		"sqlstore": newSQLiteStoreForContract(t),
	}

	for name, s := range stores {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			shortTTL, wait := kvstoreTTLParams()
			if err := s.Record(ctx, "k", shortTTL); err != nil {
				t.Fatalf("first Record: %v", err)
			}

			// Immediately Record again with a longer TTL. The entry is still
			// valid (not expired), so all implementations must treat this as a
			// no-op and NOT extend the TTL.
			if err := s.Record(ctx, "k", time.Hour); err != nil {
				t.Fatalf("second Record: %v", err)
			}

			// Wait for the original shortTTL to expire. If the second Record
			// had extended the TTL to 1h, the entry would still be valid.
			time.Sleep(wait)

			seen, err := s.Seen(ctx, "k")
			if err != nil {
				t.Fatalf("Seen: %v", err)
			}
			if seen {
				t.Fatalf(
					"Record extended the TTL (Seen=true after original TTL expiry); contract requires no-op on existing",
				)
			}
		})
	}
}

// TestKVStore_CheckAndRecord_Concurrent_AllImplementations verifies that under
// contention exactly one concurrent CheckAndRecord wins per key across every
// implementation (atomic claim, no double-wins, no lost errors).
func TestKVStore_CheckAndRecord_Concurrent_AllImplementations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	const goroutines = 50

	stores := map[string]idempotency.Store{
		"memory":   idempotency.NewMemoryStore(0),
		"kvstore":  kvstore.New(kv.NewMemStore()),
		"sqlstore": newSQLiteStoreForContract(t),
	}

	for name, s := range stores {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var wg sync.WaitGroup

			wg.Add(goroutines)

			var mu sync.Mutex

			wins, dups := 0, 0

			start := make(chan struct{})
			for i := 0; i < goroutines; i++ {
				go func() {
					defer wg.Done()
					<-start

					err := s.CheckAndRecord(ctx, "race-key", time.Minute)
					mu.Lock()
					defer mu.Unlock()

					switch {
					case err == nil:
						wins++
					case errors.Is(err, idempotency.ErrDuplicate):
						dups++
					}
				}()
			}

			close(start)
			wg.Wait()

			if wins != 1 {
				t.Fatalf("want exactly 1 winner, got %d (dups=%d)", wins, dups)
			}
			if wins+dups != goroutines {
				t.Fatalf("want %d clean outcomes, got wins=%d dups=%d", goroutines, wins, dups)
			}
		})
	}
}
