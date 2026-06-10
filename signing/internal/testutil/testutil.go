package testutil

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}

	return v
}

func CollectingPublisher() (event.PublisherFunc, *[]event.Event) {
	var published []event.Event

	pub := event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		published = append(published, events...)

		return nil
	})

	return pub, &published
}

func TrackingHandler() (func(context.Context, event.Event) error, func() bool) {
	called := false

	return func(_ context.Context, _ event.Event) error {
		called = true

		return nil
	}, func() bool { return called }
}

func NoopHandler(_ context.Context, _ event.Event) error { return nil }

func MakeTestEvent(t *testing.T) event.Event {
	t.Helper()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")

	return eventtest.NewEvent(t, "test.created", aggID, "Test", 1, []byte(`{"key":"value"}`))
}

func TamperEvent(tb testing.TB, evt event.Event) event.Event {
	tb.Helper()

	tampered, err := event.NewEvent(
		evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
		[]byte(`{"tampered":true}`),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(evt.SchemaVersion()),
		event.WithMetadata(evt.Metadata()),
	)
	if err != nil {
		tb.Fatalf("tamper event: %v", err)
	}

	return tampered
}
