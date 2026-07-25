package sqlstore_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	idemkvstore "github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// allStores builds one of every idempotency.Store implementation so contract
// tests cover the full matrix (Memory + KV + SQL), not a sample. This is the
// authoritative cross-implementation contract test; the per-module tests in
// idempotency/ and idempotency/kvstore/ are narrower unit guards.
func allStores(t *testing.T) map[string]idempotency.Store {
	t.Helper()
	return map[string]idempotency.Store{
		"memory":   idempotency.NewMemoryStore(0),
		"kvstore":  idemkvstore.New(kv.NewMemStore()),
		"sqlstore": newSQLiteStore(t),
	}
}

// TestRecordContract_AllImplementations verifies the documented Record
// contract across every idempotency.Store: Record is a no-op on an existing
// key and never extends the TTL. Closes the 2-of-3 gap (previously only Memory
// + KV were covered in idempotency/kvstore/store_test.go).
func TestRecordContract_AllImplementations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for name, s := range allStores(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Record with a short window, let it expire, then re-record with a
			// long window. Record must be a no-op on the existing key, so the
			// long TTL is NOT applied and the key stays expired. The KV store
			// regressed here once by using Set instead of SetIfAbsent.
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
				t.Fatalf("Record extended the TTL (Seen=true after expiry); contract requires no-op on existing key")
			}
		})
	}
}

// TestCheckAndRecordContract_AllImplementations verifies CheckAndRecord claims
// a fresh key and rejects the duplicate across all implementations.
func TestCheckAndRecordContract_AllImplementations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for name, s := range allStores(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := s.CheckAndRecord(ctx, "dup", time.Minute); err != nil {
				t.Fatalf("first CheckAndRecord: %v", err)
			}
			err := s.CheckAndRecord(ctx, "dup", time.Minute)
			if !errors.Is(err, idempotency.ErrDuplicate) {
				t.Fatalf("second CheckAndRecord: want ErrDuplicate, got %v", err)
			}
		})
	}
}

// TestCheckAndRecordConcurrent_AllImplementations verifies that under
// contention exactly one concurrent CheckAndRecord wins per key across every
// implementation (atomic claim, no double-wins, no lost errors).
func TestCheckAndRecordConcurrent_AllImplementations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const goroutines = 50

	for name, s := range allStores(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var wg sync.WaitGroup
			wg.Add(goroutines)

			var mu sync.Mutex
			wins, dups, other := 0, 0, 0

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
					default:
						other++
					}
				}()
			}
			close(start)
			wg.Wait()

			if wins != 1 {
				t.Fatalf("want exactly 1 winner, got %d (dups=%d other=%d)", wins, dups, other)
			}
			if wins+dups != goroutines {
				t.Fatalf("want %d clean outcomes, got wins=%d dups=%d other=%d",
					goroutines, wins, dups, other)
			}
		})
	}
}
