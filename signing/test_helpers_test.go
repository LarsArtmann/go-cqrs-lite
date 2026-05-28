package signing_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

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
