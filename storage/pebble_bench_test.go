package storage

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func BenchmarkPebbleEventStore_LoadToTimestamp_EarlyTermination(b *testing.B) {
	const totalEvents = 1000

	dir := b.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		b.Fatalf("open pebble: %v", err)
	}

	b.Cleanup(func() { _ = db.Close() })

	store := NewPebbleStore(db, slog.Default())
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

	err = store.AppendBatch(ctx, "Issue", aggID, events)
	if err != nil {
		b.Fatalf("AppendBatch: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		result, err := store.LoadToTimestamp(ctx, "Issue", aggID,
			baseTime.Add(100*time.Second))
		if err != nil {
			b.Fatalf("LoadToTimestamp: %v", err)
		}

		if len(result) != 101 {
			b.Fatalf("expected 101 events, got %d", len(result))
		}
	}
}

func BenchmarkPebbleEventStore_LoadToTimestamp_FullScan(b *testing.B) {
	const totalEvents = 1000

	dir := b.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		b.Fatalf("open pebble: %v", err)
	}

	b.Cleanup(func() { _ = db.Close() })

	store := NewPebbleStore(db, slog.Default())
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

	err = store.AppendBatch(ctx, "Issue", aggID, events)
	if err != nil {
		b.Fatalf("AppendBatch: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		result, err := store.LoadToTimestamp(ctx, "Issue", aggID,
			baseTime.Add(2000*time.Second))
		if err != nil {
			b.Fatalf("LoadToTimestamp: %v", err)
		}

		if len(result) != totalEvents {
			b.Fatalf("expected %d events, got %d", totalEvents, len(result))
		}
	}
}
