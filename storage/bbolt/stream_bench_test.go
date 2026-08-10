package bbolt

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func makeBenchEvent(tb testing.TB) (id.StreamID, event.Event) {
	tb.Helper()

	aggID := id.NewStreamID()
	evt := eventtest.NewEventOpts(
		tb,
		"test.evt",
		aggID,
		"TestAgg",
		1,
		[]byte(`{"data":"bench"}`),
	)
	return aggID, evt
}

// BenchmarkReadStreamFrom_Seek measures the O(log N) Seek-based read path
// when skipping to a specific event via the cqrs_journal_idx secondary index.
// Seeds 10K events, then iterates from event 9999 (near the end).
func BenchmarkReadStreamFrom_Seek(b *testing.B) {
	dir := b.TempDir()
	backend, err := Open(dir+"/bench.db", nil)
	if err != nil {
		b.Fatalf("open backend: %v", err)
	}
	defer func() { _ = backend.Close() }()

	store := backend.EventStore()
	ctx := context.Background()

	const totalEvents = 10_000
	var skipID id.EventID

	for range totalEvents {
		aggID, evt := makeBenchEvent(b)
		ref := id.NewStreamRef("TestAgg", aggID)
		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			b.Fatalf("Save: %v", err)
		}
		skipID = evt.ID()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		iter, err := store.ReadStreamFrom(ctx, skipID, 0)
		if err != nil {
			b.Fatalf("ReadStreamFrom: %v", err)
		}

		count := 0
		for {
			_, err := iter.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				b.Fatalf("Next: %v", err)
			}
			count++
		}
		if count > 1 {
			b.Fatalf("expected 0-1 events after last, got %d", count)
		}
		deferClose(iter)
	}
}

// BenchmarkReadStreamFrom_FullScan measures a full journal scan (no skip)
// as baseline comparison against the Seek path.
func BenchmarkReadStreamFrom_FullScan(b *testing.B) {
	dir := b.TempDir()
	backend, err := Open(dir+"/bench.db", nil)
	if err != nil {
		b.Fatalf("open backend: %v", err)
	}
	defer func() { _ = backend.Close() }()

	store := backend.EventStore()
	ctx := context.Background()

	const totalEvents = 10_000
	for range totalEvents {
		aggID, evt := makeBenchEvent(b)
		ref := id.NewStreamRef("TestAgg", aggID)
		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			b.Fatalf("Save: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		iter, err := store.ReadStreamFrom(ctx, id.EventID{}, 0)
		if err != nil {
			b.Fatalf("ReadStreamFrom: %v", err)
		}
		count := 0
		for {
			_, err := iter.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				b.Fatalf("Next: %v", err)
			}
			count++
		}
		if count != totalEvents {
			b.Fatalf("expected %d events, got %d", totalEvents, count)
		}
		deferClose(iter)
	}
}
