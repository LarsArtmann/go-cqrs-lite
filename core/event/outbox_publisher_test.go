package event

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestNewOutboxPublisher_PanicsOnNilOutbox(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()

		if r == nil {
			t.Fatal("expected panic for nil outbox")
		}

		want := "nil Outbox"
		if msg := fmt.Sprint(r); !contains(msg, want) {
			t.Fatalf("panic = %q, want containing %q", msg, want)
		}
	}()

	bus := &stubBus{}

	NewOutboxPublisher(nil, bus)
}

func TestNewOutboxPublisher_PanicsOnNilBus(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()

		if r == nil {
			t.Fatal("expected panic for nil bus")
		}

		want := "nil Bus"
		if msg := fmt.Sprint(r); !contains(msg, want) {
			t.Fatalf("panic = %q, want containing %q", msg, want)
		}
	}()

	outbox := &stubOutbox{}

	NewOutboxPublisher(outbox, nil)
}

func TestNewOutboxPublisher_Defaults(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{})

	if p.interval != time.Second {
		t.Fatalf("interval = %v, want 1s", p.interval)
	}

	if p.batchSize != 100 {
		t.Fatalf("batchSize = %d, want 100", p.batchSize)
	}
}

func TestNewOutboxPublisher_WithPollInterval(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{}, WithPollInterval(50*time.Millisecond))

	if p.interval != 50*time.Millisecond {
		t.Fatalf("interval = %v, want 50ms", p.interval)
	}
}

func TestNewOutboxPublisher_WithBatchSize(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{}, WithBatchSize(10))

	if p.batchSize != 10 {
		t.Fatalf("batchSize = %d, want 10", p.batchSize)
	}
}

func TestNewOutboxPublisher_ZeroIntervalResetsToDefault(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{}, WithPollInterval(0))

	if p.interval != time.Second {
		t.Fatalf("interval = %v, want 1s (default)", p.interval)
	}
}

func TestNewOutboxPublisher_ZeroBatchSizeResetsToDefault(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{}, WithBatchSize(0))

	if p.batchSize != 100 {
		t.Fatalf("batchSize = %d, want 100 (default)", p.batchSize)
	}
}

func TestOutboxPublisher_StartStop(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{}, WithPollInterval(10*time.Millisecond))

	err := p.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	err = p.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOutboxPublisher_DoubleStart(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{})

	err := p.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer func() { _ = p.Close() }()

	err = p.Start()
	if err == nil {
		t.Fatal("expected error on double start")
	}

	if !contains(err.Error(), "already started") {
		t.Fatalf("error = %q, want containing 'already started'", err.Error())
	}
}

func TestOutboxPublisher_CloseWithoutStart(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{})

	err := p.Close()
	if err != nil {
		t.Fatalf("Close without start: %v", err)
	}
}

func TestOutboxPublisher_DoubleClose(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{})

	_ = p.Start()
	_ = p.Close()
	_ = p.Close()
}

func TestOutboxPublisher_BackgroundPollingPublishes(t *testing.T) {
	t.Parallel()

	evt := mustNewTestEvent("user.created")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: "entry-1", Events: []Event{evt}},
	}}
	bus := &stubBus{}

	p := NewOutboxPublisher(outbox, bus, WithPollInterval(10*time.Millisecond))

	err := p.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer func() { _ = p.Close() }()

	deadline := time.After(2 * time.Second)

	for {
		bus.mu.Lock()
		published := len(bus.published)
		bus.mu.Unlock()

		if published > 0 {
			break
		}

		select {
		case <-deadline:
			t.Fatal("timed out waiting for background publish")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if len(bus.published) != 1 {
		t.Fatalf("published %d events, want 1", len(bus.published))
	}
}

func TestOutboxPublisher_PublishNow_Empty(t *testing.T) {
	t.Parallel()

	outbox := &stubOutbox{}
	bus := &stubBus{}
	p := NewOutboxPublisher(outbox, bus)

	ctx := context.Background()

	err := p.PublishNow(ctx)
	if err != nil {
		t.Fatalf("PublishNow empty: %v", err)
	}
}

func TestOutboxPublisher_PublishNow_SingleEntry(t *testing.T) {
	t.Parallel()

	evt := mustNewTestEvent("order.placed")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: "entry-1", Events: []Event{evt}},
	}}
	bus := &stubBus{}
	p := NewOutboxPublisher(outbox, bus)

	ctx := context.Background()

	err := p.PublishNow(ctx)
	if err != nil {
		t.Fatalf("PublishNow: %v", err)
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if len(bus.published) != 1 {
		t.Fatalf("published %d events, want 1", len(bus.published))
	}

	if bus.published[0].Type() != "order.placed" {
		t.Fatalf("event type = %q, want %q", bus.published[0].Type(), "order.placed")
	}

	outbox.mu.Lock()
	defer outbox.mu.Unlock()

	if len(outbox.acked) != 1 || outbox.acked[0] != "entry-1" {
		t.Fatalf("acked = %v, want [entry-1]", outbox.acked)
	}
}

func TestOutboxPublisher_PublishNow_MultipleEntries(t *testing.T) {
	t.Parallel()

	evt1 := mustNewTestEvent("user.created")
	evt2 := mustNewTestEvent("user.updated")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: "entry-1", Events: []Event{evt1}},
		{ID: "entry-2", Events: []Event{evt2}},
	}}
	bus := &stubBus{}
	p := NewOutboxPublisher(outbox, bus)

	ctx := context.Background()

	err := p.PublishNow(ctx)
	if err != nil {
		t.Fatalf("PublishNow: %v", err)
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if len(bus.published) != 2 {
		t.Fatalf("published %d events, want 2", len(bus.published))
	}

	outbox.mu.Lock()
	defer outbox.mu.Unlock()

	if len(outbox.acked) != 2 {
		t.Fatalf("acked %d entries, want 2", len(outbox.acked))
	}
}

func TestOutboxPublisher_PublishNow_PollError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("poll failed")
	outbox := &stubOutbox{pollErr: wantErr}
	bus := &stubBus{}
	p := NewOutboxPublisher(outbox, bus)

	ctx := context.Background()

	err := p.PublishNow(ctx)
	if err == nil {
		t.Fatal("expected error on poll failure")
	}

	if !contains(err.Error(), "poll pending") {
		t.Fatalf("error = %q, want containing 'poll pending'", err.Error())
	}
}

func TestOutboxPublisher_PublishNow_PublishError(t *testing.T) {
	t.Parallel()

	evt := mustNewTestEvent("user.created")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: "entry-1", Events: []Event{evt}},
	}}
	wantErr := errors.New("publish failed")
	bus := &stubBus{publishErr: wantErr}
	p := NewOutboxPublisher(outbox, bus)

	ctx := context.Background()

	err := p.PublishNow(ctx)
	if err == nil {
		t.Fatal("expected error on publish failure")
	}

	if !contains(err.Error(), "publish events") {
		t.Fatalf("error = %q, want containing 'publish events'", err.Error())
	}
}

func TestOutboxPublisher_PublishNow_PublishError_StopsAtFailure(t *testing.T) {
	t.Parallel()

	evt1 := mustNewTestEvent("user.created")
	evt2 := mustNewTestEvent("user.updated")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: "entry-1", Events: []Event{evt1}},
		{ID: "entry-2", Events: []Event{evt2}},
	}}

	callCount := 0
	bus := &stubBus{publishErrFn: func() error {
		callCount++

		if callCount > 1 {
			return errors.New("publish failed")
		}

		return nil
	}}
	p := NewOutboxPublisher(outbox, bus)

	ctx := context.Background()

	err := p.PublishNow(ctx)
	if err == nil {
		t.Fatal("expected error on second publish failure")
	}

	outbox.mu.Lock()
	defer outbox.mu.Unlock()

	if len(outbox.acked) != 1 {
		t.Fatalf("acked %d entries, want 1 (only first succeeded)", len(outbox.acked))
	}
}

func TestOutboxPublisher_PublishNow_AckError(t *testing.T) {
	t.Parallel()

	evt := mustNewTestEvent("user.created")
	outbox := &stubOutbox{
		entries: []OutboxEntry{{ID: "entry-1", Events: []Event{evt}}},
		ackErr:  errors.New("ack failed"),
	}
	bus := &stubBus{}
	p := NewOutboxPublisher(outbox, bus)

	ctx := context.Background()

	err := p.PublishNow(ctx)
	if err == nil {
		t.Fatal("expected error on ack failure")
	}

	if !contains(err.Error(), "ack entries") {
		t.Fatalf("error = %q, want containing 'ack entries'", err.Error())
	}
}

func TestOutboxPublisher_PublishNow_RespectsBatchSize(t *testing.T) {
	t.Parallel()

	p := NewOutboxPublisher(&stubOutbox{}, &stubBus{}, WithBatchSize(5))

	if p.batchSize != 5 {
		t.Fatalf("batchSize = %d, want 5", p.batchSize)
	}
}

func TestOutboxPublisher_PublishNow_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: "entry-1", Events: []Event{mustNewTestEvent("test")}},
	}}
	bus := &stubBus{}
	p := NewOutboxPublisher(outbox, bus)

	_ = p.PublishNow(ctx)
}

type stubOutbox struct {
	mu      sync.Mutex
	entries []OutboxEntry
	acked   []OutboxID
	pollErr error
	ackErr  error
}

func (o *stubOutbox) Append(_ context.Context, _ []Event) error { return nil }

func (o *stubOutbox) PollPending(_ context.Context, limit int) ([]OutboxEntry, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.pollErr != nil {
		return nil, o.pollErr
	}

	if len(o.entries) > limit {
		return o.entries[:limit], nil
	}

	return o.entries, nil
}

func (o *stubOutbox) Ack(_ context.Context, ids []OutboxID) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.ackErr != nil {
		return o.ackErr
	}

	o.acked = append(o.acked, ids...)

	return nil
}

func (o *stubOutbox) Close() error { return nil }

type stubBus struct {
	mu           sync.Mutex
	published    []Event
	publishErr   error
	publishErrFn func() error
}

func (b *stubBus) Publish(_ context.Context, events ...Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.publishErrFn != nil {
		err := b.publishErrFn()
		if err != nil {
			return err
		}
	}

	if b.publishErr != nil {
		return b.publishErr
	}

	b.published = append(b.published, events...)

	return nil
}

func (b *stubBus) Subscribe(_ Type, _ Handler) error { return nil }
func (b *stubBus) SubscribeAll(_ Handler) error      { return nil }
func (b *stubBus) Use(_ ...Middleware) error         { return nil }
func (b *stubBus) Close() error                      { return nil }

func mustNewTestEvent(eventType Type) Event {
	evt, err := NewEvent(
		eventType,
		id.MustParseAggregateID("01HGW5FPJPYK5RE8ACZDesWMY2"),
		"TestAggregate",
		1,
		[]byte(`{"key":"value"}`),
	)
	if err != nil {
		panic(fmt.Sprintf("mustNewTestEvent: %v", err))
	}

	return evt
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
