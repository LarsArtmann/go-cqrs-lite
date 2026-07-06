package idempotency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v3"
)

// testCmd is a minimal command.Command implementation for middleware tests.
type testCmd struct {
	id  id.CommandID
	typ command.Type
	agg id.AggregateID
}

func (c *testCmd) Type() command.Type          { return c.typ }
func (c *testCmd) AggregateID() id.AggregateID { return c.agg }
func (c *testCmd) ID() id.CommandID            { return c.id }

func newTestCmd() *testCmd {
	return &testCmd{
		id:  id.NewCommandID(),
		typ: "test.do",
		agg: id.NewAggregateID(),
	}
}

func TestCommandIdempotency_FirstCallPassesThrough(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var called bool
	mw := idempotency.CommandIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true

		return nil
	})

	cmd := newTestCmd()
	err := handler(context.Background(), cmd)
	if err != nil {
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
	mw := idempotency.CommandIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	cmd := newTestCmd()
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
	mw := idempotency.CommandIdempotency(store, time.Minute, nil)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	ctx := context.Background()
	_ = handler(ctx, newTestCmd())
	_ = handler(ctx, newTestCmd()) // different ID → not a duplicate

	if callCount != 2 {
		t.Fatalf("handler call count: want 2, got %d", callCount)
	}
}

func TestCommandIdempotency_CustomKeyExtractor(t *testing.T) {
	t.Parallel()

	store := idempotency.NewMemoryStore(0)
	defer store.Close()

	var callCount int
	mw := idempotency.CommandIdempotency(
		store, time.Minute,
		func(cmd command.Command) string { return "fixed-key" },
	)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	ctx := context.Background()
	_ = handler(ctx, newTestCmd())
	err := handler(ctx, newTestCmd()) // different command, but same fixed key

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
	mw := idempotency.CommandIdempotency(
		store, time.Minute,
		func(_ command.Command) string { return "" }, // empty → skip dedup
	)
	handler := mw(func(_ context.Context, _ command.Command) error {
		callCount++

		return nil
	})

	ctx := context.Background()
	_ = handler(ctx, newTestCmd())
	_ = handler(ctx, newTestCmd()) // empty key → no dedup → both pass

	if callCount != 2 {
		t.Fatalf("handler call count: want 2 (empty key skips dedup), got %d", callCount)
	}
}

func TestCommandIdempotency_StoreErrorIsTransient(t *testing.T) {
	t.Parallel()

	store := &failingStore{}
	mw := idempotency.CommandIdempotency(store, time.Minute, nil)

	var called bool
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true

		return nil
	})

	err := handler(context.Background(), newTestCmd())
	if called {
		t.Fatal("handler must NOT be called when store errors")
	}
	if fam := errorfamily.Classify(err); fam != errorfamily.Transient {
		t.Fatalf("store error family: want Transient, got %s", fam)
	}
}

func TestCommandIDKey_ReturnsCommandIDString(t *testing.T) {
	t.Parallel()

	cmd := newTestCmd()
	key := idempotency.CommandIDKey(cmd)
	if key != cmd.ID().String() {
		t.Fatalf("CommandIDKey: want %s, got %s", cmd.ID().String(), key)
	}
	if key == "" {
		t.Fatal("CommandIDKey returned empty string for a valid command")
	}
}

// failingStore is a Store that always returns an error on CheckAndRecord.
type failingStore struct{}

func (failingStore) Seen(context.Context, string) (bool, error)          { return false, nil }
func (failingStore) Record(context.Context, string, time.Duration) error { return nil }
func (failingStore) CheckAndRecord(context.Context, string, time.Duration) error {
	return errorfamily.NewInfrastructure("test.store_failure", "simulated store failure")
}
