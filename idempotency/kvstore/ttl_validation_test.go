package kvstore_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"
)

// TestStore_NonPositiveTTLRejected_AllStores verifies the TTL-validation
// contract across every production implementation (memory, kvstore, sqlstore):
// Record and CheckAndRecord reject a zero or negative TTL with ErrInvalidTTL,
// and the rejected key is never recorded. This mirrors the MemoryStore contract
// test in go-idempotency and closes the audit gap from
// docs/status/2026-08-07_22-38 for the kvstore and sqlstore adapters.
func TestStore_NonPositiveTTLRejected_AllStores(t *testing.T) {
	t.Parallel()

	for name, factory := range allStores() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store, cleanup := factory(t)
			defer cleanup()

			ctx := context.Background()

			invalidTTLs := []struct {
				name string
				ttl  time.Duration
			}{
				{"zero", 0},
				{"negative_nanosecond", -time.Nanosecond},
				{"negative_hour", -time.Hour},
			}

			for _, tc := range invalidTTLs {
				key := "ttl-rejected/" + tc.name

				if err := store.Record(ctx, key, tc.ttl); !errors.Is(err, idempotency.ErrInvalidTTL) {
					t.Errorf("Record(%s): want ErrInvalidTTL, got %v", tc.name, err)
				}

				claimKey := "ttl-rejected-claim/" + tc.name
				if err := store.CheckAndRecord(ctx, claimKey, tc.ttl); !errors.Is(err, idempotency.ErrInvalidTTL) {
					t.Errorf("CheckAndRecord(%s): want ErrInvalidTTL, got %v", tc.name, err)
				}

				assertRejectedKeyUnseen(t, store, key)
				assertRejectedKeyUnseen(t, store, claimKey)
			}
		})
	}
}

// assertRejectedKeyUnseen fails the test when a key rejected for an invalid
// TTL is nonetheless reported as seen (validation must precede any write).
func assertRejectedKeyUnseen(t *testing.T, store idempotency.Store, key string) {
	t.Helper()

	seen, err := store.Seen(context.Background(), key)
	if err != nil {
		t.Fatalf("Seen(%q): %v", key, err)
	}

	if seen {
		t.Fatalf("key %q rejected for invalid TTL must not be Seen", key)
	}
}
