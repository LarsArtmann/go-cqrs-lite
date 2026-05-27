package storage

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

var errTestPoller = errors.New("test poller error")

type fakePollerOutbox struct {
	mu        sync.Mutex
	entries   []event.OutboxEntry
	pollErr   error
	ackErr    error
	ackedIDs  []event.OutboxID
	pollCalls int
}

func (o *fakePollerOutbox) Close() error                                    { return nil }
func (o *fakePollerOutbox) Append(_ context.Context, _ []event.Event) error { return nil }

func (o *fakePollerOutbox) PollPending(_ context.Context, limit int) ([]event.OutboxEntry, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.pollCalls++
	if o.pollErr != nil {
		return nil, o.pollErr
	}

	ackedSet := make(map[event.OutboxID]struct{}, len(o.ackedIDs))
	for _, id := range o.ackedIDs {
		ackedSet[id] = struct{}{}
	}

	var pending []event.OutboxEntry
	for _, entry := range o.entries {
		if _, ok := ackedSet[entry.ID]; !ok {
			pending = append(pending, entry)
		}
	}

	if len(pending) > limit {
		pending = pending[:limit]
	}

	return pending, nil
}

func (o *fakePollerOutbox) Ack(_ context.Context, ids []event.OutboxID) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.ackErr != nil {
		return o.ackErr
	}
	o.ackedIDs = append(o.ackedIDs, ids...)
	return nil
}

func (o *fakePollerOutbox) PollCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.pollCalls
}

func (o *fakePollerOutbox) AckedIDs() []event.OutboxID {
	o.mu.Lock()
	defer o.mu.Unlock()
	result := make([]event.OutboxID, len(o.ackedIDs))
	copy(result, o.ackedIDs)
	return result
}

type fakePollerPublisher struct {
	mu         sync.Mutex
	published  []event.Event
	publishErr error
}

func (p *fakePollerPublisher) Publish(_ context.Context, events ...event.Event) error {
	if p.publishErr != nil {
		return p.publishErr
	}
	p.mu.Lock()
	p.published = append(p.published, events...)
	p.mu.Unlock()
	return nil
}

func (p *fakePollerPublisher) Published() []event.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.published
}

func newPollerTestEvent(t *testing.T, eventType string, aggID id.AggregateID, version event.Version) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type(eventType),
		aggID,
		"User",
		version,
		[]byte(`{"name":"test"}`),
		event.WithOccurredAt(time.Now().Truncate(time.Microsecond)),
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

func TestOutboxPoller_PollAndPublish(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt := newPollerTestEvent(t, "UserCreated", aggID, 1)

	outbox := &fakePollerOutbox{
		entries: []event.OutboxEntry{
			{ID: "outbox-1", Events: []event.Event{evt}},
		},
	}
	publisher := &fakePollerPublisher{}

	poller := NewOutboxPoller(outbox, publisher, WithPollInterval(10*time.Millisecond))

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	poller.Start(ctx)
	<-ctx.Done()
	poller.Stop()

	if outbox.PollCount() == 0 {
		t.Fatal("expected at least one poll call")
	}

	if len(publisher.Published()) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.Published()))
	}

	if len(outbox.ackedIDs) != 1 || outbox.ackedIDs[0] != "outbox-1" {
		t.Fatalf("expected ack outbox-1, got %v", outbox.ackedIDs)
	}
}

func TestOutboxPoller_PollError(t *testing.T) {
	t.Parallel()

	outbox := &fakePollerOutbox{pollErr: errTestPoller}
	publisher := &fakePollerPublisher{}

	poller := NewOutboxPoller(outbox, publisher, WithPollInterval(10*time.Millisecond))

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	poller.Start(ctx)
	<-ctx.Done()
	poller.Stop()

	if outbox.PollCount() == 0 {
		t.Fatal("expected at least one poll call")
	}

	if len(publisher.Published()) != 0 {
		t.Fatalf("expected 0 published events, got %d", len(publisher.Published()))
	}
}

func TestOutboxPoller_PublishError_SkipsAck(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt := newPollerTestEvent(t, "UserCreated", aggID, 1)

	outbox := &fakePollerOutbox{
		entries: []event.OutboxEntry{
			{ID: "outbox-1", Events: []event.Event{evt}},
		},
	}
	publisher := &fakePollerPublisher{publishErr: errTestPoller}

	poller := NewOutboxPoller(outbox, publisher)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	poller.Start(ctx)
	<-ctx.Done()
	poller.Stop()

	if len(publisher.Published()) != 0 {
		t.Fatalf("expected 0 published events, got %d", len(publisher.Published()))
	}

	if len(outbox.ackedIDs) != 0 {
		t.Fatalf("expected 0 acked IDs, got %v", outbox.ackedIDs)
	}
}

func TestOutboxPoller_AckError(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt := newPollerTestEvent(t, "UserCreated", aggID, 1)

	outbox := &fakePollerOutbox{
		entries: []event.OutboxEntry{
			{ID: "outbox-1", Events: []event.Event{evt}},
		},
		ackErr: errTestPoller,
	}
	publisher := &fakePollerPublisher{}

	poller := NewOutboxPoller(outbox, publisher, WithPollInterval(10*time.Millisecond))

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	poller.Start(ctx)
	<-ctx.Done()
	poller.Stop()

	if len(publisher.Published()) < 1 {
		t.Fatalf("expected at least 1 published event, got %d", len(publisher.Published()))
	}
}

func TestOutboxPoller_EmptyEntries(t *testing.T) {
	t.Parallel()

	outbox := &fakePollerOutbox{entries: []event.OutboxEntry{}}
	publisher := &fakePollerPublisher{}

	poller := NewOutboxPoller(outbox, publisher, WithPollInterval(10*time.Millisecond))

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	poller.Start(ctx)
	<-ctx.Done()
	poller.Stop()

	if len(publisher.Published()) != 0 {
		t.Fatalf("expected 0 published events, got %d", len(publisher.Published()))
	}

	if len(outbox.ackedIDs) != 0 {
		t.Fatalf("expected 0 acked IDs, got %v", outbox.ackedIDs)
	}
}

func TestOutboxPoller_MultipleEventsPerEntry(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt1 := newPollerTestEvent(t, "UserCreated", aggID, 1)
	evt2 := newPollerTestEvent(t, "UserUpdated", aggID, 2)

	outbox := &fakePollerOutbox{
		entries: []event.OutboxEntry{
			{ID: "outbox-1", Events: []event.Event{evt1, evt2}},
		},
	}
	publisher := &fakePollerPublisher{}

	poller := NewOutboxPoller(outbox, publisher, WithPollInterval(10*time.Millisecond))

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	poller.Start(ctx)
	<-ctx.Done()
	poller.Stop()

	if len(publisher.Published()) != 2 {
		t.Fatalf("expected 2 published events, got %d", len(publisher.Published()))
	}

	if acked := outbox.AckedIDs(); len(acked) != 1 {
		t.Fatalf("expected 1 acked entry, got %d", len(acked))
	}
}

func TestOutboxPoller_PartialPublish_SkipsFailedEntry(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt1 := newPollerTestEvent(t, "UserCreated", aggID, 1)
	evt2 := newPollerTestEvent(t, "UserUpdated", aggID, 2)

	outbox := &fakePollerOutbox{
		entries: []event.OutboxEntry{
			{ID: "outbox-1", Events: []event.Event{evt1}},
			{ID: "outbox-2", Events: []event.Event{evt2}},
		},
	}

	failAfter := 1
	callCount := 0
	publisher := &fakePollerPublisher{
		publishErr: errTestPoller,
	}

	// Override to fail after first successful publish
	wrappedPublisher := &fakePublisherWrapper{
		inner: publisher,
		fn: func(p *fakePollerPublisher, events []event.Event) error {
			callCount++
			if callCount > failAfter {
				return errTestPoller
			}
			p.mu.Lock()
			p.published = append(p.published, events...)
			p.mu.Unlock()
			return nil
		},
	}

	poller := NewOutboxPoller(outbox, wrappedPublisher, WithPollInterval(10*time.Millisecond))

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	poller.Start(ctx)
	<-ctx.Done()
	poller.Stop()

	if len(publisher.Published()) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.Published()))
	}

	if acked := outbox.AckedIDs(); len(acked) != 1 || acked[0] != "outbox-1" {
		t.Fatalf("expected only outbox-1 acked, got %v", acked)
	}
}

type fakePublisherWrapper struct {
	inner *fakePollerPublisher
	fn    func(*fakePollerPublisher, []event.Event) error
}

func (p *fakePublisherWrapper) Publish(_ context.Context, events ...event.Event) error {
	return p.fn(p.inner, events)
}

func TestOutboxPoller_Options(t *testing.T) {
	t.Parallel()

	outbox := &fakePollerOutbox{}
	publisher := &fakePollerPublisher{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	poller := NewOutboxPoller(
		outbox,
		publisher,
		WithPollInterval(2*time.Second),
		WithBatchSize(50),
		WithPollerLogger(logger),
	)

	if poller.interval != 2*time.Second {
		t.Fatalf("expected interval 2s, got %v", poller.interval)
	}

	if poller.batchSize != 50 {
		t.Fatalf("expected batchSize 50, got %d", poller.batchSize)
	}
}
