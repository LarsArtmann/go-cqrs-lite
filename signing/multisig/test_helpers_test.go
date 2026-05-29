package multisig_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func collectingPublisher() (event.PublisherFunc, *[]event.Event) {
	var published []event.Event

	pub := event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		published = append(published, events...)

		return nil
	})

	return pub, &published
}

func trackingHandler() (func(context.Context, event.Event) error, func() bool) {
	called := false

	return func(_ context.Context, _ event.Event) error {
		called = true

		return nil
	}, func() bool { return called }
}

func noopHandler(_ context.Context, _ event.Event) error { return nil }

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

func assertPanicMessage(t *testing.T, fn func(), expectedMsg string) {
	t.Helper()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic with message %q, got no panic", expectedMsg)
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}

		if msg != expectedMsg {
			t.Fatalf("expected panic message %q, got %q", expectedMsg, msg)
		}
	}()

	fn()
}
