package event

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func assertErrorContains(t *testing.T, err error, substr string) {
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error = %q, want containing %q", err.Error(), substr)
	}
}

func TestOutboxPublisher_BackgroundPollingPublishes(t *testing.T) {
	t.Parallel()

	evt := mustNewTestEvent("user.created")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: NewOutboxID("entry-1"), Events: []Event{evt}},
	}}
	bus := &stubPublisher{}

	p, err := NewOutboxPublisher(outbox, bus, WithPollInterval(10*time.Millisecond))
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	err = p.Start()
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
	bus := &stubPublisher{}

	p, err := NewOutboxPublisher(outbox, bus)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	ctx := context.Background()

	err = p.PublishNow(ctx)
	if err != nil {
		t.Fatalf("PublishNow empty: %v", err)
	}
}

func TestOutboxPublisher_PublishNow_SingleEntry(t *testing.T) {
	t.Parallel()

	evt := mustNewTestEvent("order.placed")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: NewOutboxID("entry-1"), Events: []Event{evt}},
	}}
	bus := &stubPublisher{}

	p, err := NewOutboxPublisher(outbox, bus)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	ctx := context.Background()

	err = p.PublishNow(ctx)
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

	if len(outbox.acked) != 1 || !outbox.acked[0].Equal(NewOutboxID("entry-1")) {
		t.Fatalf("acked = %v, want [entry-1]", outbox.acked)
	}
}

func TestOutboxPublisher_PublishNow_MultipleEntries(t *testing.T) {
	t.Parallel()

	evt1 := mustNewTestEvent("user.created")
	evt2 := mustNewTestEvent("user.updated")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: NewOutboxID("entry-1"), Events: []Event{evt1}},
		{ID: NewOutboxID("entry-2"), Events: []Event{evt2}},
	}}
	bus := &stubPublisher{}

	p, err := NewOutboxPublisher(outbox, bus)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	ctx := context.Background()

	err = p.PublishNow(ctx)
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
	bus := &stubPublisher{}

	p, err := NewOutboxPublisher(outbox, bus)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	ctx := context.Background()

	err = p.PublishNow(ctx)
	if err == nil {
		t.Fatal("expected error on poll failure")
	}

	assertErrorContains(t, err, "poll pending")
}

func TestOutboxPublisher_PublishNow_PublishError(t *testing.T) {
	t.Parallel()

	evt := mustNewTestEvent("user.created")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: NewOutboxID("entry-1"), Events: []Event{evt}},
	}}
	wantErr := errors.New("publish failed")
	bus := &stubPublisher{publishErr: wantErr}

	p, err := NewOutboxPublisher(outbox, bus)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	ctx := context.Background()

	err = p.PublishNow(ctx)
	if err == nil {
		t.Fatal("expected error on publish failure")
	}

	assertErrorContains(t, err, "publish events")
}

func TestOutboxPublisher_PublishNow_PublishError_StopsAtFailure(t *testing.T) {
	t.Parallel()

	evt1 := mustNewTestEvent("user.created")
	evt2 := mustNewTestEvent("user.updated")
	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: NewOutboxID("entry-1"), Events: []Event{evt1}},
		{ID: NewOutboxID("entry-2"), Events: []Event{evt2}},
	}}

	callCount := 0
	bus := &stubPublisher{publishErrFn: func() error {
		callCount++

		if callCount > 1 {
			return errors.New("publish failed")
		}

		return nil
	}}

	p, err := NewOutboxPublisher(outbox, bus)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	ctx := context.Background()

	err = p.PublishNow(ctx)
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
		entries: []OutboxEntry{{ID: NewOutboxID("entry-1"), Events: []Event{evt}}},
		ackErr:  errors.New("ack failed"),
	}
	bus := &stubPublisher{}

	p, err := NewOutboxPublisher(outbox, bus)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	ctx := context.Background()

	err = p.PublishNow(ctx)
	if err == nil {
		t.Fatal("expected error on ack failure")
	}

	assertErrorContains(t, err, "ack outbox entries")
}

func TestOutboxPublisher_PublishNow_RespectsBatchSize(t *testing.T) {
	t.Parallel()

	p := newOutboxPublisherWithBatchSize(t, 5)

	if p.batchSize != 5 {
		t.Fatalf("batchSize = %d, want 5", p.batchSize)
	}
}

func TestOutboxPublisher_PublishNow_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	outbox := &stubOutbox{entries: []OutboxEntry{
		{ID: NewOutboxID("entry-1"), Events: []Event{mustNewTestEvent("test")}},
	}}
	bus := &stubPublisher{}

	p, err := NewOutboxPublisher(outbox, bus)
	if err != nil {
		t.Fatalf("NewOutboxPublisher: %v", err)
	}

	err = p.PublishNow(ctx)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}
