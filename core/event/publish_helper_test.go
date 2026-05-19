package event_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type mockPublisher struct {
	published []event.Event
	err       error
}

func (m *mockPublisher) Publish(_ context.Context, events ...event.Event) error {
	m.published = append(m.published, events...)

	return m.err
}

type mockOutbox struct {
	events []event.Event
	err    error
}

func (m *mockOutbox) Append(_ context.Context, events []event.Event) error {
	m.events = append(m.events, events...)

	return m.err
}

func (m *mockOutbox) PollPending(_ context.Context, _ int) ([]event.OutboxEntry, error) {
	return nil, nil
}

func (m *mockOutbox) Ack(_ context.Context, _ []event.OutboxID) error {
	return nil
}

func (m *mockOutbox) Close() error { return nil }

func testEvents(t *testing.T, n int) []event.Event {
	t.Helper()

	events := make([]event.Event, n)

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	for i := range n {
		evt, err := event.NewEvent("TestEvent", aggID, "Test", event.Version(i+1), nil)
		if err != nil {
			t.Fatalf("create test event %d: %v", i, err)
		}

		events[i] = evt
	}

	return events
}

func TestPublishChanges_DirectPublish(t *testing.T) {
	t.Parallel()

	pub := &mockPublisher{}
	events := testEvents(t, 2)

	err := event.PublishChanges(context.Background(), pub, nil, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pub.published) != 2 {
		t.Errorf("expected 2 published events, got %d", len(pub.published))
	}
}

func TestPublishChanges_OutboxAppend(t *testing.T) {
	t.Parallel()

	pub := &mockPublisher{}
	outbox := &mockOutbox{}
	events := testEvents(t, 2)

	err := event.PublishChanges(context.Background(), pub, outbox, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pub.published) != 0 {
		t.Errorf("expected 0 published events (should use outbox), got %d", len(pub.published))
	}

	if len(outbox.events) != 2 {
		t.Errorf("expected 2 outbox events, got %d", len(outbox.events))
	}
}

func TestPublishChanges_PublishError(t *testing.T) {
	t.Parallel()

	pub := &mockPublisher{err: errTestPublish}
	events := testEvents(t, 1)

	err := event.PublishChanges(context.Background(), pub, nil, events)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublishChanges_OutboxError(t *testing.T) {
	t.Parallel()

	outbox := &mockOutbox{err: errTestOutbox}
	events := testEvents(t, 1)

	err := event.PublishChanges(context.Background(), nil, outbox, events)
	if err == nil {
		t.Fatal("expected error")
	}
}

var (
	errTestPublish = errors.New("publish failed")
	errTestOutbox  = errors.New("outbox failed")
)
