package bbolt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// BenchmarkEventDeserialize measures the per-event decode cost of the bbolt
// read path (CBOR envelope → event.Event): the dominant CPU/allocation source
// when loading streams. Compare before/after changes to deserializeEvent, and
// against storage/pebble's BenchmarkEventDeserialize for cross-backend parity.
func BenchmarkEventDeserialize(b *testing.B) {
	evt, err := event.NewEvent("UserCreated", id.NewStreamID(), "User", event.Version(1),
		[]byte(`{"name":"Alice","email":"alice@example.com","age":42}`),
		event.WithCorrelationID(id.NewCorrelationID()),
		event.WithCausationID(id.NewCausationID()),
		event.WithUserID(id.NewUserID()),
	)
	if err != nil {
		b.Fatal(err)
	}

	data, err := serializeEvent(evt)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := deserializeEvent(data); err != nil {
			b.Fatal(err)
		}
	}
}
