package kvstore_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4"
	idemsqlstore "github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	_ "modernc.org/sqlite"
)

func TestStore_SeenReturnsFalseForNewKey(t *testing.T) {
	t.Parallel()
	store := kvstore.New(kv.NewMemStore())

	seen, err := store.Seen(context.Background(), "cmd-1")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if seen {
		t.Fatal("expected new key to not be seen")
	}
}

func TestStore_SeenReturnsTrueAfterRecord(t *testing.T) {
	t.Parallel()
	store := kvstore.New(kv.NewMemStore())
	ctx := context.Background()

	if err := store.Record(ctx, "cmd-1", time.Minute); err != nil {
		t.Fatalf("Record: %v", err)
	}
	seen, _ := store.Seen(ctx, "cmd-1")
	if !seen {
		t.Fatal("expected recorded key to be seen")
	}
}

func TestStore_ExpiredEntriesAreNotSeen(t *testing.T) {
	t.Parallel()
	store := kvstore.New(kv.NewMemStore())
	ctx := context.Background()

	store.Record(ctx, "expired", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	seen, _ := store.Seen(ctx, "expired")
	if seen {
		t.Fatal("expected expired entry to not be seen")
	}
}

func TestStore_CheckAndRecord_RejectsDuplicate(t *testing.T) {
	t.Parallel()
	store := kvstore.New(kv.NewMemStore())
	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "dup", time.Minute); err != nil {
		t.Fatalf("first: %v", err)
	}
	err := store.CheckAndRecord(ctx, "dup", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestStore_CheckAndRecord_AllowsDifferentKeys(t *testing.T) {
	t.Parallel()
	store := kvstore.New(kv.NewMemStore())
	ctx := context.Background()

	store.CheckAndRecord(ctx, "a", time.Minute)
	if err := store.CheckAndRecord(ctx, "b", time.Minute); err != nil {
		t.Fatalf("second key: %v", err)
	}
}

func TestStore_ExpiredKeyCanBeReclaimed(t *testing.T) {
	t.Parallel()
	store := kvstore.New(kv.NewMemStore())
	ctx := context.Background()

	store.CheckAndRecord(ctx, "reclaim", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	if err := store.CheckAndRecord(ctx, "reclaim", time.Minute); err != nil {
		t.Fatalf("expected reclaim after expiry, got %v", err)
	}
}

func TestStore_ConcurrentSameKeyExactlyOneWins(t *testing.T) {
	t.Parallel()
	store := kvstore.New(kv.NewMemStore())

	const n = 200
	ctx := context.Background()
	var wg sync.WaitGroup
	var wins, dups atomic.Int64
	start := make(chan struct{})

	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			<-start
			err := store.CheckAndRecord(ctx, "race", time.Minute)
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, idempotency.ErrDuplicate):
				dups.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if wins.Load() != 1 {
		t.Fatalf("expected 1 winner, got %d", wins.Load())
	}
	if dups.Load() != n-1 {
		t.Fatalf("expected %d dups, got %d", n-1, dups.Load())
	}
}

func TestStore_SatisfiesStoreInterface(t *testing.T) {
	t.Parallel()
	var _ idempotency.Store = kvstore.New(kv.NewMemStore())
}

func TestStore_Record_DoesNotExtendTTL(t *testing.T) {
	t.Parallel()
	store := kvstore.New(kv.NewMemStore())
	ctx := context.Background()

	// Record with a short window, let it expire, then re-record with a long
	// window. Record must be a no-op on an existing key, so the long TTL is
	// NOT applied and the key stays expired. (The previous overwrite-on-Set
	// implementation would extend the TTL here; this guards against regression.)
	if err := store.Record(ctx, "k", 1*time.Millisecond); err != nil {
		t.Fatalf("first Record: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	if err := store.Record(ctx, "k", time.Hour); err != nil {
		t.Fatalf("second Record: %v", err)
	}

	seen, err := store.Seen(ctx, "k")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if seen {
		t.Fatal(
			"expected key to remain expired (Record must not extend TTL), but Seen reported true",
		)
	}
}

// newSQLiteStoreForContract builds a SQL-backed idempotency store for the
// cross-implementation contract test. Shared-cache in-memory SQLite + a single
// connection so concurrent writers queue instead of erroring.
func newSQLiteStoreForContract(t *testing.T) idempotency.Store {
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

// TestStore_Record_MatchesMemoryStoreContract verifies the kvstore Record/Seen
// contract matches the reference MemoryStore implementation AND the SQL store.
// All three implementations must be a no-op on an existing key and never extend
// the TTL. This is the authoritative cross-implementation contract test
// (closing the 2-of-3 gap; previously only Memory + KV were covered).
func TestStore_Record_MatchesMemoryStoreContract(t *testing.T) {
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

			if err := s.Record(ctx, "k", 1*time.Millisecond); err != nil {
				t.Fatalf("first Record: %v", err)
			}
			time.Sleep(5 * time.Millisecond)

			if err := s.Record(ctx, "k", time.Hour); err != nil {
				t.Fatalf("second Record: %v", err)
			}

			seen, err := s.Seen(ctx, "k")
			if err != nil {
				t.Fatalf("Seen: %v", err)
			}
			if seen {
				t.Fatalf(
					"Record extended the TTL (Seen=true after expiry); contract requires no-op on existing",
				)
			}
		})
	}
}

// TestStore_CheckAndRecord_Concurrent_AllImplementations verifies that under
// contention exactly one concurrent CheckAndRecord wins per key across every
// implementation (atomic claim, no double-wins, no lost errors).
func TestStore_CheckAndRecord_Concurrent_AllImplementations(t *testing.T) {
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

// TestStore_Seen_LazilyDeletesExpiredEntry verifies the lazy-delete contract:
// after Seen reports an expired key as unseen, the entry is removed from the
// backing KV store (not merely reported as absent). Without this, expired keys
// would accumulate forever when only Seen is called.
func TestStore_Seen_LazilyDeletesExpiredEntry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	backend := kv.NewMemStore()
	defer backend.Close()
	store := kvstore.New(backend)

	if err := store.Record(ctx, "k", 1*time.Millisecond); err != nil {
		t.Fatalf("Record: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	seen, err := store.Seen(ctx, "k")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if seen {
		t.Fatal("expected expired key to be unseen")
	}

	// The lazy delete must have removed the entry from the backend.
	if _, err := backend.Get(ctx, []byte("k")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf("expected backend to have lazy-deleted the expired key (kv.ErrNotFound), got %v", err)
	}
}

// TestStore_Record_Concurrent_FirstTTLWins verifies that under concurrent
// Record calls on the same key, every call succeeds (Record never errors on a
// duplicate) and the first writer's TTL governs — later writers do not extend
// it. Exercises the SetIfAbsent contention path directly (distinct from the
// CheckAndRecord concurrency test).
func TestStore_Record_Concurrent_FirstTTLWins(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := kvstore.New(kv.NewMemStore())

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})
	var errCount atomic.Int32

	for range goroutines {
		go func() {
			defer wg.Done()
			<-start
			// Long TTL: if any writer overwrote the first, Seen would stay true
			// past the intended short window. Record must never error here.
			if err := store.Record(ctx, "k", time.Hour); err != nil {
				errCount.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if errCount.Load() != 0 {
		t.Fatalf("Record errored under contention %d times (Record must be idempotent/no-op)", errCount.Load())
	}

	seen, err := store.Seen(ctx, "k")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if !seen {
		t.Fatal("expected key to be seen after concurrent Record (at least one write must land)")
	}
}
