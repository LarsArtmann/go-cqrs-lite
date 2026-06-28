package stack_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
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

func TestBundle_GracefulClose_CompletesQuickly(t *testing.T) {
	t.Parallel()

	b, err := stack.New(stack.WithReadModels(kv.NewMemStore()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.GracefulClose(ctx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}
}

func TestBundle_GracefulClose_TimesOut(t *testing.T) {
	t.Parallel()

	block := make(chan struct{})
	b, err := stack.New(
		stack.WithReadModels(kv.NewMemStore()),
		stack.WithCloser(&channelCloser{block: block}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := b.GracefulClose(ctx); err == nil {
		t.Fatal("GracefulClose should timeout with blocking closer")
	}

	close(block) // unblock the goroutine
}

func TestBundle_GracefulClose_DrainsBeforeClose(t *testing.T) {
	t.Parallel()

	drainer := &trackingDrainer{}
	b, err := stack.New(
		stack.WithReadModels(kv.NewMemStore()),
		stack.WithDrainer(drainer),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.GracefulClose(ctx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}

	if !drainer.drained {
		t.Fatal("Drain was not called before Close")
	}
}

func TestBundle_GracefulClose_DrainErrorAbortsClose(t *testing.T) {
	t.Parallel()

	drainErr := errors.New("drain failed")
	drainer := &trackingDrainer{err: drainErr}
	closer := &trackingCloser{}
	b, err := stack.New(
		stack.WithReadModels(kv.NewMemStore()),
		stack.WithDrainer(drainer),
		stack.WithCloser(closer),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = b.GracefulClose(ctx)
	if !errors.Is(err, drainErr) {
		t.Fatalf("expected drain error, got: %v", err)
	}

	if closer.closed {
		t.Fatal("Close should NOT be called when Drain fails")
	}
}

func TestBundle_GracefulClose_DrainersCalledInOrder(t *testing.T) {
	t.Parallel()

	first := &orderedDrainer{name: "first"}
	second := &orderedDrainer{name: "second"}
	b, err := stack.New(
		stack.WithReadModels(kv.NewMemStore()),
		stack.WithDrainer(first),
		stack.WithDrainer(second),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.GracefulClose(ctx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}

	if first.seq >= second.seq {
		t.Fatal("drainers should be called in registration order")
	}
}

// trackingDrainer records that Drain was called.
type trackingDrainer struct {
	drained bool
	err     error
}

func (d *trackingDrainer) Drain(_ context.Context) error {
	d.drained = true

	return d.err
}

// trackingCloser records that Close was called.
type trackingCloser struct{ closed bool }

func (c *trackingCloser) Close() error {
	c.closed = true

	return nil
}

// orderedDrainer records the order it was drained via a shared counter.
type orderedDrainer struct {
	name string
	seq  int
}

var drainCounter int

func (d *orderedDrainer) Drain(_ context.Context) error {
	drainCounter++
	d.seq = drainCounter

	return nil
}

// channelCloser blocks on Close until the channel is closed.
// Deterministic — no time.Sleep.
type channelCloser struct {
	block chan struct{}
}

func (c *channelCloser) Close() error {
	<-c.block

	return nil
}

func TestBundle_Debug_ShowsCapabilities(t *testing.T) {
	t.Parallel()

	b, err := stack.New(
		stack.WithReadModels(kv.NewMemStore()),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	output := b.Debug()

	if !strings.Contains(output, "ReadModels") {
		t.Errorf("Debug() output missing ReadModels:\n%s", output)
	}

	if !strings.Contains(output, "EventSink") {
		t.Errorf("Debug() output missing EventSink:\n%s", output)
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
