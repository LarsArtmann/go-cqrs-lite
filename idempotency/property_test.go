package idempotency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
)

// TestProperty_RecordIsIdempotent: Recording the same key multiple times
// never errors and the key remains seen.
func TestProperty_RecordIsIdempotent(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		store := idempotency.NewMemoryStore(0)
		defer store.Close()

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

// TestProperty_CheckAndRecordExactlyOnce: Among N concurrent CheckAndRecord
// calls for the same key, exactly one succeeds and the rest get ErrDuplicate.
func TestProperty_CheckAndRecordExactlyOnce(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		store := idempotency.NewMemoryStore(0)
		defer store.Close()

		key := rapid.String().Draw(t, "key")
		n := rapid.IntRange(2, 20).Draw(t, "concurrent")
		ttl := time.Minute

		results := make(chan error, n)
		for i := 0; i < n; i++ {
			go func() {
				results <- store.CheckAndRecord(context.Background(), key, ttl)
			}()
		}

		successes := 0
		duplicates := 0
		for i := 0; i < n; i++ {
			err := <-results
			switch {
			case err == nil:
				successes++
			case errors.Is(err, idempotency.ErrDuplicate):
				duplicates++
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}

		if successes != 1 {
			t.Fatalf("expected exactly 1 success, got %d (of %d)", successes, n)
		}
		if duplicates != n-1 {
			t.Fatalf("expected %d duplicates, got %d", n-1, duplicates)
		}
	})
}

// TestProperty_KeysAreIndependent: Operations on one key don't affect another.
func TestProperty_KeysAreIndependent(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		store := idempotency.NewMemoryStore(0)
		defer store.Close()

		keyA := rapid.StringMatching(`.+`).Draw(t, "keyA")
		keyB := rapid.StringMatching(`.+`).Filter(func(s string) bool { return s != keyA }).Draw(t, "keyB")
		ttl := time.Minute

		if err := store.Record(context.Background(), keyA, ttl); err != nil {
			t.Fatalf("Record A: %v", err)
		}

		seenB, err := store.Seen(context.Background(), keyB)
		if err != nil {
			t.Fatalf("Seen B: %v", err)
		}
		if seenB {
			t.Fatal("keyB should not be seen (only keyA was recorded)")
		}

		if err := store.CheckAndRecord(context.Background(), keyB, ttl); err != nil {
			t.Fatalf("CheckAndRecord B (first call): %v", err)
		}

		seenA, err := store.Seen(context.Background(), keyA)
		if err != nil {
			t.Fatalf("Seen A: %v", err)
		}
		if !seenA {
			t.Fatal("keyA should still be seen after operating on keyB")
		}
	})
}

// TestProperty_TTLExpiry: After TTL expires, Seen returns false and
// CheckAndRecord succeeds again.
func TestProperty_TTLExpiry(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		store := idempotency.NewMemoryStore(0)
		defer store.Close()

		key := rapid.String().Draw(t, "key")
		ttl := 50 * time.Millisecond

		if err := store.CheckAndRecord(context.Background(), key, ttl); err != nil {
			t.Fatalf("CheckAndRecord (first): %v", err)
		}

		time.Sleep(ttl + 20*time.Millisecond)

		seen, err := store.Seen(context.Background(), key)
		if err != nil {
			t.Fatalf("Seen after TTL: %v", err)
		}
		if seen {
			t.Fatal("key should not be seen after TTL expiry")
		}

		if err := store.CheckAndRecord(context.Background(), key, ttl); err != nil {
			t.Fatalf("CheckAndRecord after TTL expiry should succeed: %v", err)
		}
	})
}
