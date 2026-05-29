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

	err := event.PublishChanges(context.Background(), pub, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pub.published) != 2 {
		t.Errorf("expected 2 published events, got %d", len(pub.published))
	}
}

func TestPublishChanges_PublishError(t *testing.T) {
	t.Parallel()

	pub := &mockPublisher{err: errTestPublish}
	events := testEvents(t, 1)

	err := event.PublishChanges(context.Background(), pub, events)
	if err == nil {
		t.Fatal("expected error")
	}
}

var errTestPublish = errors.New("publish failed")
