package pebble

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func seedPebbleBenchEvents(
	b *testing.B,
	totalEvents int,
) (*EventStore, id.AggregateID, time.Time) {
	b.Helper()

	dir := b.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{}) //nolint:varnamelen
	if err != nil {
		b.Fatalf("open pebble: %v", err)
	}

	b.Cleanup(func() { _ = db.Close() })

	store := NewStore(db, slog.Default())
	aggID := id.NewAggregateID()
	ctx := context.Background()
	baseTime := time.Now()

	events := make([]event.Event, totalEvents)
	for i := range totalEvents {
		evt, err := event.NewEvent(
			"IssueCreated", aggID, "Issue", event.Version(i+1),
			[]byte(fmt.Sprintf(`{"title":"test-%d"}`, i+1)),
			event.WithOccurredAt(baseTime.Add(time.Duration(i)*time.Second)),
		)
		if err != nil {
			b.Fatalf("create event %d: %v", i, err)
		}

		events[i] = evt
	}

	err = store.AppendBatch(ctx, event.NewAggregateRef(event.AggregateType("Issue"), aggID), events)
	if err != nil {
		b.Fatalf("AppendBatch: %v", err)
	}

	return store, aggID, baseTime
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
			store, aggID, baseTime := seedPebbleBenchEvents(b, 1000)
			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				result, err := store.LoadToTimestamp(
					ctx,
					event.NewAggregateRef(event.AggregateType("Issue"), aggID),
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
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("BenchEvent", aggID, "Bench", event.Version(1),
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
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent("BenchEvent", aggID, "Bench", event.Version(1),
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

	return NewStore(database, slog.Default())
}
