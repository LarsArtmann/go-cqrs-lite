package idempotency_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/idempotency/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

func TestKVStore_SeenReturnsFalseForNewKey(t *testing.T) {
	t.Parallel()
	store := idempotency.NewKVStore(kv.NewMemStore())

	seen, err := store.Seen(context.Background(), "cmd-1")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if seen {
		t.Fatal("expected new key to not be seen")
	}
}

func TestKVStore_SeenReturnsTrueAfterRecord(t *testing.T) {
	t.Parallel()
	store := idempotency.NewKVStore(kv.NewMemStore())
	ctx := context.Background()

	if err := store.Record(ctx, "cmd-1", time.Minute); err != nil {
		t.Fatalf("Record: %v", err)
	}
	seen, _ := store.Seen(ctx, "cmd-1")
	if !seen {
		t.Fatal("expected recorded key to be seen")
	}
}

func TestKVStore_ExpiredEntriesAreNotSeen(t *testing.T) {
	t.Parallel()
	store := idempotency.NewKVStore(kv.NewMemStore())
	ctx := context.Background()

	store.Record(ctx, "expired", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	seen, _ := store.Seen(ctx, "expired")
	if seen {
		t.Fatal("expected expired entry to not be seen")
	}
}

func TestKVStore_CheckAndRecord_RejectsDuplicate(t *testing.T) {
	t.Parallel()
	store := idempotency.NewKVStore(kv.NewMemStore())
	ctx := context.Background()

	if err := store.CheckAndRecord(ctx, "dup", time.Minute); err != nil {
		t.Fatalf("first: %v", err)
	}
	err := store.CheckAndRecord(ctx, "dup", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestKVStore_CheckAndRecord_AllowsDifferentKeys(t *testing.T) {
	t.Parallel()
	store := idempotency.NewKVStore(kv.NewMemStore())
	ctx := context.Background()

	store.CheckAndRecord(ctx, "a", time.Minute)
	if err := store.CheckAndRecord(ctx, "b", time.Minute); err != nil {
		t.Fatalf("second key: %v", err)
	}
}

func TestKVStore_ExpiredKeyCanBeReclaimed(t *testing.T) {
	t.Parallel()
	store := idempotency.NewKVStore(kv.NewMemStore())
	ctx := context.Background()

	store.CheckAndRecord(ctx, "reclaim", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	if err := store.CheckAndRecord(ctx, "reclaim", time.Minute); err != nil {
		t.Fatalf("expected reclaim after expiry, got %v", err)
	}
}

func TestKVStore_ConcurrentSameKeyExactlyOneWins(t *testing.T) {
	t.Parallel()
	store := idempotency.NewKVStore(kv.NewMemStore())

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

func TestKVStore_SatisfiesStoreInterface(t *testing.T) {
	t.Parallel()
	var _ idempotency.Store = idempotency.NewKVStore(kv.NewMemStore())
}
