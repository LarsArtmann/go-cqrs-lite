package pebble

import (
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// BenchmarkEventDeserialize measures the per-event decode cost of the Pebble
// read path (CBOR envelope → event.Event): the dominant CPU/allocation source
// when loading streams. Compare before/after changes to deserializeEvent.
func BenchmarkEventDeserialize(b *testing.B) {
	db, err := pebble.Open(b.TempDir(), &pebble.Options{})
	if err != nil {
		b.Fatalf("open pebble: %v", err)
	}

	defer func() { _ = db.Close() }()

	store, err := NewStore(db, slog.Default())
	if err != nil {
		b.Fatal(err)
	}

	evt, err := event.NewEvent("UserCreated", id.NewStreamID(), "User", event.Version(1),
		[]byte(`{"name":"Alice","email":"alice@example.com","age":42}`),
		event.WithCorrelationID(id.NewCorrelationID()),
		event.WithCausationID(id.NewCausationID()),
		event.WithUserID(id.NewUserID()),
	)
	if err != nil {
		b.Fatal(err)
	}

	data, err := store.serializeEvent(evt)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := store.deserializeEvent(data); err != nil {
			b.Fatal(err)
		}
	}
}
