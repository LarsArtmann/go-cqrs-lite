package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-idempotency"
)

// Middleware TTL-expiry coverage: the store-level suites prove entries expire,
// but nothing previously verified the END-TO-END behavior a consumer depends
// on — a duplicate command rejected within the TTL becomes processable again
// after the TTL lapses (retry-after-outage semantics).

// idempotencyTTLTestParams returns (ttl, wait) for middleware TTL tests. The
// wait clears the TTL by a 5x margin so the test holds under -race, where
// scheduling latency inflates 5-10x, without a build-tagged constant pair.
func idempotencyTTLTestParams() (time.Duration, time.Duration) {
	return 100 * time.Millisecond, 500 * time.Millisecond
}

// TestCommandIdempotency_KeyExpiresAfterTTL dispatches the same command three
// times: first passes, duplicate within the TTL returns ErrDuplicate without
// invoking the handler, and after the TTL expires the command is processable
// again (handler called exactly twice in total).
func TestCommandIdempotency_KeyExpiresAfterTTL(t *testing.T) {
	t.Parallel()

	store := newTestIdempotencyStore(t)
	defer store.Close()

	ttl, wait := idempotencyTTLTestParams()

	var callCount int
	mw := CommandIdempotency(store, ttl, nil)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	cmd := newIdempotencyTestCmd()
	ctx := context.Background()

	if err := handler(ctx, cmd); err != nil {
		t.Fatalf("first dispatch: want nil, got %v", err)
	}

	if err := handler(ctx, cmd); !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("duplicate within TTL: want ErrDuplicate, got %v", err)
	}

	time.Sleep(wait)

	if err := handler(ctx, cmd); err != nil {
		t.Fatalf("dispatch after TTL expiry: want nil (key reclaimable), got %v", err)
	}

	if callCount != 2 {
		t.Fatalf("handler call count: want 2 (initial + post-expiry), got %d", callCount)
	}
}
