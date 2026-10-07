package kvstore_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"

	"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// ttlTestParams returns a (ttl, wait) pair with enough headroom to survive
// -race detector scheduling inflation. The base values are deliberately large
// enough that clock resolution jitter on CI VMs cannot flip the expiry check.
func ttlTestParams() (time.Duration, time.Duration) {
	if raceEnabled {
		return 100 * time.Millisecond, 400 * time.Millisecond
	}

	return 10 * time.Millisecond, 50 * time.Millisecond
}

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

	ttl, wait := ttlTestParams()
	store.Record(ctx, "expired", ttl)
	time.Sleep(wait)

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

	ttl, wait := ttlTestParams()
	store.CheckAndRecord(ctx, "reclaim", ttl)
	time.Sleep(wait)

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
	shortTTL, wait := ttlTestParams()
	if err := store.Record(ctx, "k", shortTTL); err != nil {
		t.Fatalf("first Record: %v", err)
	}
	time.Sleep(wait)

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

// The 3-way cross-implementation contract tests (Record no-op on an existing
// key, concurrent CheckAndRecord exactly-once) and the all-stores rapid
// property suite live in integration/idempotency_contract_test.go and
// integration/idempotency_property_test.go: they need the SQLite-backed
// store, and keeping them there drops the SQLite dependency from this module.

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

	shortTTL, wait := ttlTestParams()
	if err := store.Record(ctx, "k", shortTTL); err != nil {
		t.Fatalf("Record: %v", err)
	}
	time.Sleep(wait)

	seen, err := store.Seen(ctx, "k")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if seen {
		t.Fatal("expected expired key to be unseen")
	}

	// The lazy delete must have removed the entry from the backend.
	if _, err := backend.Get(ctx, []byte("k")); !errors.Is(err, kv.ErrNotFound) {
		t.Fatalf(
			"expected backend to have lazy-deleted the expired key (kv.ErrNotFound), got %v",
			err,
		)
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
		t.Fatalf(
			"Record errored under contention %d times (Record must be idempotent/no-op)",
			errCount.Load(),
		)
	}

	seen, err := store.Seen(ctx, "k")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if !seen {
		t.Fatal("expected key to be seen after concurrent Record (at least one write must land)")
	}
}
