package stack_test

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
)

// compile-time check: the FakeStore from eventtest satisfies event.Store.
// (eventtest.FakeStore is a pointer type.)

// countingCloser tracks how many times Close was called, for dedup and
// rollback tests.
type countingCloser struct {
	count atomic.Int32
}

func (c *countingCloser) Close() error {
	c.count.Add(1)

	return nil
}

var _ io.Closer = (*countingCloser)(nil)

func TestNew_AssignsFields(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	readModels := kv.NewMemStore()

	b, err := stack.New(
		stack.WithEventStore(store),
		stack.WithBus(bus),
		stack.WithReadModels(readModels),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if b.EventSink == nil {
		t.Fatal("EventSink not set")
	}

	if b.EventSource == nil {
		t.Fatal("EventSource not set")
	}

	if b.Publisher == nil {
		t.Fatal("Publisher not set")
	}

	if b.Subscriber == nil {
		t.Fatal("Subscriber not set")
	}

	if b.ReadModels == nil {
		t.Fatal("ReadModels not set")
	}
}

func TestNew_WithEventStore_SetsJournalWhenSupported(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()

	b, err := stack.New(stack.WithEventStore(store))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if b.Journal == nil {
		t.Fatal("Journal not set despite store implementing it")
	}

	if b.SeekableJournal == nil {
		t.Fatal("SeekableJournal not set despite store implementing it")
	}
}

func TestNew_EmptyBundleReturnsErrEmpty(t *testing.T) {
	t.Parallel()

	_, err := stack.New()
	if !errors.Is(err, stack.ErrEmpty) {
		t.Fatalf("got err=%v, want ErrEmpty", err)
	}
}

func TestNew_ValidationFailure_ClosesRegisteredClosers(t *testing.T) {
	t.Parallel()

	tracker := &countingCloser{}

	// WithCloser registers a closer without setting any capability field,
	// so the bundle is empty and validation fails. New must close the
	// tracker before returning the error.
	_, err := stack.New(stack.WithCloser(tracker))
	if !errors.Is(err, stack.ErrEmpty) {
		t.Fatalf("got err=%v, want ErrEmpty", err)
	}

	if got := tracker.count.Load(); got != 1 {
		t.Fatalf("tracker closed %d times, want 1 (rollback)", got)
	}
}

func TestBundle_Close_DeduplicatesSamePointer(t *testing.T) {
	t.Parallel()

	tracker := &countingCloser{}

	// Register the same closer twice; Close should call it once.
	b, err := stack.New(
		stack.WithReadModels(kv.NewMemStore()),
		stack.WithCloser(tracker),
		stack.WithCloser(tracker),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if got := tracker.count.Load(); got != 1 {
		t.Fatalf("tracker closed %d times, want 1 (pointer dedup)", got)
	}
}

func TestBundle_Close_Idempotent(t *testing.T) {
	t.Parallel()

	b, err := stack.New(stack.WithReadModels(kv.NewMemStore()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// ── Accessor tests ──

type testState struct{ Count int }

type stateKey string

func (k stateKey) String() string { return string(k) }

func TestReadModel_Accessor_ProducesWorkingStore(t *testing.T) {
	t.Parallel()

	b, err := stack.New(stack.WithReadModels(kv.NewMemStore()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	type record struct {
		Value string `json:"value"`
	}

	store, err := stack.ReadModel[record, stateKey](b, codec.JSONCodec{})
	if err != nil {
		t.Fatalf("ReadModel: %v", err)
	}

	ctx := context.Background()

	if err := store.Set(ctx, "1", &record{Value: "hello"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Value != "hello" {
		t.Fatalf("got %q, want %q", got.Value, "hello")
	}
}

func TestReadModel_Accessor_MissingBackendReturnsError(t *testing.T) {
	t.Parallel()

	b, err := stack.New(stack.WithEventStore(eventtest.NewFakeStore()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = stack.ReadModel[testState, stateKey](b, codec.JSONCodec{})
	if !errors.Is(err, stack.ErrMissingReadModels) {
		t.Fatalf("got err=%v, want ErrMissingReadModels", err)
	}
}

func TestReadModel_Accessor_NilCodecDefaultsToJSON(t *testing.T) {
	t.Parallel()

	b, err := stack.New(stack.WithReadModels(kv.NewMemStore()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	type record struct {
		Value string `json:"value"`
	}

	store, err := stack.ReadModel[record, stateKey](b, nil)
	if err != nil {
		t.Fatalf("ReadModel: %v", err)
	}

	ctx := context.Background()

	if err := store.Set(ctx, "1", &record{Value: "ok"}); err != nil {
		t.Fatalf("Set with default JSON codec: %v", err)
	}
}
