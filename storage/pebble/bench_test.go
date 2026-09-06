package pebble

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func seedPebbleBenchEvents(
	b *testing.B,
	totalEvents int,
) (*EventStore, id.StreamID, time.Time) {
	b.Helper()

	dir := b.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		b.Fatalf("open pebble: %v", err)
	}

	b.Cleanup(func() { _ = db.Close() })

	store, err := NewStore(db, slog.Default())
	if err != nil {
		b.Fatal(err)
	}
	streamID := id.NewStreamID()
	ctx := context.Background()
	baseTime := time.Now()

	events := make([]event.Event, totalEvents)
	for i := range totalEvents {
		evt, err := event.NewEvent(
			"IssueCreated", streamID, "Issue", event.Version(i+1),
			[]byte(fmt.Sprintf(`{"title":"test-%d"}`, i+1)),
			event.WithOccurredAt(baseTime.Add(time.Duration(i)*time.Second)),
		)
		if err != nil {
			b.Fatalf("create event %d: %v", i, err)
		}

		events[i] = evt
	}

	err = store.AppendBatch(ctx, id.NewStreamRef(id.StreamType("Issue"), streamID), events)
	if err != nil {
		b.Fatalf("AppendBatch: %v", err)
	}

	return store, streamID, baseTime
}

func BenchmarkEventStore_LoadToTimestamp(b *testing.B) {
	b.ReportAllocs()
	tests := []struct {
		name     string
		offset   time.Duration
		expected int
	}{
		{"EarlyTermination", 100 * time.Second, 101},
		{"FullScan", 2000 * time.Second, 1000},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			store, streamID, baseTime := seedPebbleBenchEvents(b, 1000)
			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				result, err := store.LoadToTimestamp(
					ctx,
					id.NewStreamRef(id.StreamType("Issue"), streamID),
					baseTime.Add(tc.offset),
				)
				if err != nil {
					b.Fatalf("LoadToTimestamp: %v", err)
				}

				if len(result) != tc.expected {
					b.Fatalf("expected %d events, got %d", tc.expected, len(result))
				}
			}
		})
	}
}

func BenchmarkSerializeEnvelope(b *testing.B) {
	b.ReportAllocs()

	store := newPebbleBenchStore(b)
	streamID := id.NewStreamID()

	evt, err := event.NewEvent("BenchEvent", streamID, "Bench", event.Version(1),
		[]byte(`{"name":"benchmark","value":42}`))
	if err != nil {
		b.Fatalf("create event: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := store.serializeEvent(evt)
		if err != nil {
			b.Fatalf("serialize: %v", err)
		}
	}
}

func BenchmarkDeserializeEnvelope(b *testing.B) {
	b.ReportAllocs()

	store := newPebbleBenchStore(b)
	streamID := id.NewStreamID()

	evt, err := event.NewEvent("BenchEvent", streamID, "Bench", event.Version(1),
		[]byte(`{"name":"benchmark","value":42}`))
	if err != nil {
		b.Fatalf("create event: %v", err)
	}

	data, err := store.serializeEvent(evt)
	if err != nil {
		b.Fatalf("serialize: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := store.deserializeEvent(data)
		if err != nil {
			b.Fatalf("deserialize: %v", err)
		}
	}
}

func newPebbleBenchStore(b *testing.B) *EventStore {
	b.Helper()

	dir := b.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		b.Fatalf("open pebble: %v", err)
	}

	b.Cleanup(func() { _ = database.Close() })

	store, err := NewStore(database, slog.Default())
	if err != nil {
		b.Fatal(err)
	}

	return store
}

func BenchmarkEventStore_Save_SingleEvent(b *testing.B) {
	b.ReportAllocs()

	store := newPebbleBenchStore(b)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		streamID := id.NewStreamID()
		ref := id.NewStreamRef("Bench", streamID)

		evt, err := event.NewEvent("BenchSaved", streamID, "Bench", event.Version(1),
			[]byte(`{"name":"save-bench","value":42}`))
		if err != nil {
			b.Fatalf("create event: %v", err)
		}

		err = store.Save(ctx, ref, []event.Event{evt}, event.Version(0))
		if err != nil {
			b.Fatalf("Save: %v", err)
		}
	}
}

func BenchmarkEventStore_AppendBatch_10Events(b *testing.B) {
	b.ReportAllocs()

	store := newPebbleBenchStore(b)
	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		streamID := id.NewStreamID()
		ref := id.NewStreamRef("Bench", streamID)

		events := make([]event.Event, 10)
		for idx := range 10 {
			evt, err := event.NewEvent("BenchBatch", streamID, "Bench", event.Version(idx+1),
				[]byte(`{"name":"batch","idx":`+strconv.Itoa(idx)+`}`))
			if err != nil {
				b.Fatalf("create event %d: %v", idx, err)
			}

			events[idx] = evt
		}

		err := store.AppendBatch(ctx, ref, events)
		if err != nil {
			b.Fatalf("AppendBatch: %v", err)
		}
	}
}
