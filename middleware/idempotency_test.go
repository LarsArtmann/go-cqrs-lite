package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func newIdempotencyTestCmd() *testCommand {
	return &testCommand{
		commandID:   id.NewCommandID(),
		streamID: id.NewAggregateID(),
	}
}

// --- Command ---

func TestCommandIdempotency_FirstCallPassesThrough(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var called bool
	mw := CommandIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true

		return nil
	})

	if err := handler(context.Background(), newIdempotencyTestCmd()); err != nil {
		t.Fatalf("first dispatch: want nil, got %v", err)
	}
	if !called {
		t.Fatal("handler was not called on first dispatch")
	}
}

func TestCommandIdempotency_DuplicateReturnsErrDuplicate(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := CommandIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	cmd := newIdempotencyTestCmd()
	ctx := context.Background()
	_ = handler(ctx, cmd)
	err := handler(ctx, cmd) // same command → same ID → duplicate

	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second dispatch: want ErrDuplicate, got %v", err)
	}
	if callCount != 1 {
		t.Fatalf("handler call count: want 1, got %d", callCount)
	}
}

func TestCommandIdempotency_DifferentCommandsBothPass(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := CommandIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	ctx := context.Background()
	_ = handler(ctx, newIdempotencyTestCmd())
	_ = handler(ctx, newIdempotencyTestCmd()) // different ID → not a duplicate

	if callCount != 2 {
		t.Fatalf("handler call count: want 2, got %d", callCount)
	}
}

func TestCommandIdempotency_CustomKeyExtractor(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := CommandIdempotency(
		store, time.Minute,
		func(cmd command.Command) string { return "fixed-key" },
	)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	ctx := context.Background()
	_ = handler(ctx, newIdempotencyTestCmd())
	err := handler(ctx, newIdempotencyTestCmd()) // different command, same fixed key

	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second dispatch with fixed key: want ErrDuplicate, got %v", err)
	}
	if callCount != 1 {
		t.Fatalf("handler call count: want 1, got %d", callCount)
	}
}

func TestCommandIdempotency_EmptyKeyPassesThrough(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := CommandIdempotency(
		store, time.Minute,
		func(_ command.Command) string { return "" }, // empty → skip dedup
	)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	ctx := context.Background()
	_ = handler(ctx, newIdempotencyTestCmd())
	_ = handler(ctx, newIdempotencyTestCmd()) // empty key → no dedup → both pass

	if callCount != 2 {
		t.Fatalf("handler call count: want 2 (empty key skips dedup), got %d", callCount)
	}
}

func TestCommandIdempotency_StoreErrorIsTransient(t *testing.T) {
	t.Parallel()

	store := &failingIdempotencyStore{}
	mw := CommandIdempotency(store, time.Minute, nil)

	var called bool
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true

		return nil
	})

	err := handler(context.Background(), newIdempotencyTestCmd())
	if called {
		t.Fatal("handler must NOT be called when store errors")
	}
	if fam := errorfamily.Classify(err); fam != errorfamily.Transient {
		t.Fatalf("store error family: want Transient, got %s", fam)
	}
}

// --- Event ---

func TestEventIdempotency_FirstCallPassesThrough(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var called bool
	mw := EventIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ event.Event) error {
		called = true

		return nil
	})

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("NewTestEvent: %v", err)
	}

	if err := handler(context.Background(), evt); err != nil {
		t.Fatalf("first handle: want nil, got %v", err)
	}
	if !called {
		t.Fatal("handler was not called on first handle")
	}
}

func TestEventIdempotency_DuplicateReturnsErrDuplicate(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := EventIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ event.Event) error {
		callCount++

		return nil
	})

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("NewTestEvent: %v", err)
	}

	ctx := context.Background()
	_ = handler(ctx, evt)
	err = handler(ctx, evt) // same event → same ID → duplicate

	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second handle: want ErrDuplicate, got %v", err)
	}
	if callCount != 1 {
		t.Fatalf("handler call count: want 1, got %d", callCount)
	}
}

func TestEventIdempotency_DifferentEventsBothPass(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := EventIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ event.Event) error {
		callCount++

		return nil
	})

	ctx := context.Background()
	for range 2 {
		evt, err := eventtest.NewTestEvent()
		if err != nil {
			t.Fatalf("NewTestEvent: %v", err)
		}
		if err := handler(ctx, evt); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}

	if callCount != 2 {
		t.Fatalf("handler call count: want 2, got %d", callCount)
	}
}

// --- Query ---

func TestQueryIdempotency_DeduplicatesByKey(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := QueryIdempotency(
		store, time.Minute,
		func(_ query.Query) string { return "fixed-query-key" },
	)
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		callCount++

		return "result", nil
	})

	ctx := context.Background()
	q := &testQuery{}
	result, err := handler(ctx, q)
	if err != nil {
		t.Fatalf("first query: %v", err)
	}
	if result != "result" {
		t.Fatalf("first result: want \"result\", got %v", result)
	}

	result2, err := handler(ctx, q) // same key → duplicate
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second query: want ErrDuplicate, got %v", err)
	}
	if result2 != nil {
		t.Fatalf("second result: want nil, got %v", result2)
	}
	if callCount != 1 {
		t.Fatalf("handler call count: want 1, got %d", callCount)
	}
}

func TestQueryIdempotency_EmptyKeyPassesThrough(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := QueryIdempotency(
		store, time.Minute,
		func(_ query.Query) string { return "" }, // empty → skip dedup
	)
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		callCount++

		return "result", nil
	})

	ctx := context.Background()
	q := &testQuery{}
	_, _ = handler(ctx, q)
	_, _ = handler(ctx, q) // empty key → no dedup → both pass

	if callCount != 2 {
		t.Fatalf("handler call count: want 2 (empty key skips dedup), got %d", callCount)
	}
}

// --- Helpers ---

type failingIdempotencyStore struct{}

func (failingIdempotencyStore) Seen(
	context.Context,
	string,
) (bool, error) {
	return false, nil
}
func (failingIdempotencyStore) Record(context.Context, string, time.Duration) error { return nil }

func (failingIdempotencyStore) CheckAndRecord(context.Context, string, time.Duration) error {
	return errorfamily.NewInfrastructure("test.store_failure", "simulated store failure")
}
