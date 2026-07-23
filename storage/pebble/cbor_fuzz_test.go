package pebble

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func FuzzDeserializeEvent(f *testing.F) {
	dir := f.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		f.Fatalf("open pebble: %v", err)
	}

	f.Cleanup(func() { _ = database.Close() })

	store, err := NewStore(database, slog.Default())
	if err != nil {
		f.Fatal(err)
	}

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("FuzzEvent", aggID, "Fuzz", event.Version(1),
		[]byte(`{"key":"value"}`))
	if err != nil {
		f.Fatalf("create event: %v", err)
	}

	validCBOR, err := store.serializeEvent(evt)
	if err != nil {
		f.Fatalf("serialize: %v", err)
	}

	f.Add(validCBOR)
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xff})
	f.Add([]byte(`{"id":"test"}`))
	f.Add([]byte{0xa0})
	f.Add([]byte{0xbf, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := store.deserializeEvent(data)
		if err != nil {
			return
		}
	})
}

func FuzzSerializeDeserializeRoundtrip(f *testing.F) {
	payloads := [][]byte{
		[]byte(`{"name":"Alice"}`),
		[]byte(``),
		make([]byte, 256),
		[]byte(`{"nested":{"deep":true},"arr":[1,2,3]}`),
	}

	for _, p := range payloads {
		f.Add(p)
	}

	f.Fuzz(func(t *testing.T, payload []byte) {
		t.Parallel()

		store := newPebbleTestStore(t)
		aggID := id.NewAggregateID()

		evt, err := event.NewEvent("FuzzEvent", aggID, "Fuzz", event.Version(1), payload)
		if err != nil {
			t.Fatalf("create event: %v", err)
		}

		data, err := store.serializeEvent(evt)
		if err != nil {
			t.Fatalf("serialize: %v", err)
		}

		got, err := store.deserializeEvent(data)
		if err != nil {
			t.Fatalf("deserialize: %v", err)
		}

		if got.Type() != evt.Type() {
			t.Errorf("type mismatch: want %s, got %s", evt.Type(), got.Type())
		}

		if got.Version() != evt.Version() {
			t.Errorf("version mismatch: want %d, got %d", evt.Version(), got.Version())
		}

		if got.StreamID() != evt.StreamID() {
			t.Error("aggregate ID mismatch")
		}

		if !bytes.Equal(event.PayloadReadOnly(got), payload) {
			t.Error("payload mismatch after round-trip")
		}
	})
}
