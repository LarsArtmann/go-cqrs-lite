package signing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

// trackingHandler returns an event handler and a function to check if it was called.
func trackingHandler() (func(context.Context, event.Event) error, func() bool) {
	called := false

	return func(_ context.Context, _ event.Event) error {
		called = true

		return nil
	}, func() bool { return called }
}

// noopHandler returns an event handler that does nothing.
func noopHandler(_ context.Context, _ event.Event) error { return nil }

// makeTestEvent creates a deterministic event for signing tests.
func makeTestEvent(t *testing.T) event.Event {
	t.Helper()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	return testhelpers.NewEvent(t, "test.created", aggID, "Test", 1, []byte(`{"key":"value"}`))
}

func tamperEvent(tb testing.TB, evt event.Event) event.Event {
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
