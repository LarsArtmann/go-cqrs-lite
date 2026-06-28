package idempotency_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v3"
)

func TestMemoryStore_CheckAndRecord_FirstCallSucceeds(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	err := store.CheckAndRecord(context.Background(), "cmd-1", time.Minute)
	if err != nil {
		t.Fatalf("first CheckAndRecord: want nil, got %v", err)
	}
}

func TestMemoryStore_CheckAndRecord_DuplicateReturnsErrDuplicate(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	if err := store.CheckAndRecord(ctx, "cmd-1", time.Minute); err != nil {
		t.Fatalf("first call: %v", err)
	}

	err := store.CheckAndRecord(ctx, "cmd-1", time.Minute)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second call: want ErrDuplicate, got %v", err)
	}

	// ErrDuplicate must be a Conflict so it maps to HTTP 409 downstream.
	if fam := event.Classify(err); fam != event.Conflict {
		t.Fatalf("family: want Conflict, got %s", fam)
	}
}

func TestMemoryStore_CheckAndRecord_AtomicUnderConcurrency(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	const goroutines = 200
	const key = "contended-cmd"

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		wins    int
		dups    int
		started = make(chan struct{})
	)

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()

			<-started // release all goroutines at once

			err := store.CheckAndRecord(context.Background(), key, time.Minute)
			mu.Lock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, idempotency.ErrDuplicate):
				dups++
			default:
				t.Errorf("unexpected error: %v", err)
			}
			mu.Unlock()
		}()
	}

	close(started)
	wg.Wait()

	if wins != 1 {
		t.Fatalf("wins: want exactly 1 winner, got %d", wins)
	}
	if dups != goroutines-1 {
		t.Fatalf("dups: want %d, got %d", goroutines-1, dups)
	}
}

func TestMemoryStore_Seen_AfterRecord(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()

	seen, err := store.Seen(ctx, "cmd-2")
	if err != nil || seen {
		t.Fatalf("before record: want seen=false, nil; got %v, %v", seen, err)
	}

	_ = store.Record(ctx, "cmd-2", time.Minute)

	seen, err = store.Seen(ctx, "cmd-2")
	if err != nil || !seen {
		t.Fatalf("after record: want seen=true, nil; got %v, %v", seen, err)
	}
}

func TestMemoryStore_Seen_LazilyDeletesExpired(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0) // no sweep goroutine
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "cmd-3", 20*time.Millisecond)

	time.Sleep(40 * time.Millisecond)

	seen, err := store.Seen(ctx, "cmd-3")
	if err != nil || seen {
		t.Fatalf("after TTL: want seen=false, got %v, %v", seen, err)
	}

	// The lazy delete must have removed the entry.
	seen, err = store.Seen(ctx, "cmd-3")
	if err != nil || seen {
		t.Fatalf("after lazy delete: want seen=false, got %v, %v", seen, err)
	}
}

func TestMemoryStore_Record_NoopOnExistingKey(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "cmd-4", time.Minute)

	// A second Record must not extend the TTL (no-op).
	_ = store.Record(ctx, "cmd-4", 5*time.Minute)

	// Still seen — no-op doesn't clear it.
	seen, _ := store.Seen(ctx, "cmd-4")
	if !seen {
		t.Fatal("Record no-op cleared a valid key")
	}
}

func TestMemoryStore_Sweep_RemovesExpiredEntries(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(15 * time.Millisecond)
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "cmd-5", 10*time.Millisecond)

	// Wait beyond the sweep interval so the background goroutine runs at least once.
	time.Sleep(60 * time.Millisecond)

	seen, _ := store.Seen(ctx, "cmd-5")
	if seen {
		t.Fatal("sweep did not remove expired entry")
	}
}

func TestMemoryStore_Close_Idempotent(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(time.Hour)
	store.Close()
	store.Close() // must not panic
}

func TestMemoryStore_CheckAndRecord_AllowsAfterExpiry(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	ctx := context.Background()
	_ = store.CheckAndRecord(ctx, "cmd-6", 20*time.Millisecond)

	// Within TTL: duplicate.
	if err := store.CheckAndRecord(
		ctx,
		"cmd-6",
		time.Minute,
	); !errors.Is(
		err,
		idempotency.ErrDuplicate,
	) {
		t.Fatalf("within TTL: want ErrDuplicate, got %v", err)
	}

	time.Sleep(40 * time.Millisecond)

	// After expiry: a fresh recording is allowed.
	if err := store.CheckAndRecord(ctx, "cmd-6", time.Minute); err != nil {
		t.Fatalf("after expiry: want nil, got %v", err)
	}
}

func TestErrDuplicate_IsConflict(t *testing.T) {
	t.Parallel()

	if fam := event.Classify(idempotency.ErrDuplicate); fam != event.Conflict {
		t.Fatalf("family: want Conflict, got %s", fam)
	}
	if event.IsRetryable(idempotency.ErrDuplicate) {
		t.Fatal("Conflict must not be retryable")
	}
}
