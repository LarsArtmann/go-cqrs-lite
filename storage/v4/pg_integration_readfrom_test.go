//go:build integration

package storage_test

// PostgreSQL integration tests for journal keyset pagination (ReadFrom).
// Run with: go test -tags=integration ./...
//
// These tests verify the two-step keyset pagination (cursor point lookup +
// timestamp-range query) against real PostgreSQL: $N placeholder numbering,
// time.Time round-trip through the occurred_at TIMESTAMP column, and
// (occurred_at, id) tie-break ordering.

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func pgJournalStore(t *testing.T) *storage.SQLEventStore {
	t.Helper()

	backend, err := storage.NewSQLBackend(pgDB(t))
	if err != nil {
		t.Fatalf("NewSQLBackend: %v", err)
	}

	return backend.EventStore()
}

func TestPostgresEventStore_ReadFrom_KeysetEquivalence(t *testing.T) {
	store := pgJournalStore(t)
	ctx := context.Background()

	cfg := eventtest.IssueStoreConfig()
	base := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)

	const total = 24 // 8 tie-groups of 3

	events := make([]event.Event, total)
	aggIDs := make([]id.StreamID, total)

	for i := 0; i < total; i++ {
		aggIDs[i] = id.NewStreamID()
		events[i] = cfg.NewTestEvent(t, aggIDs[i], 1,
			event.WithOccurredAt(base.Add(time.Duration(i/3)*time.Millisecond)))
	}

	// Insert newest-first so physical row order diverges from journal order.
	for i := total - 1; i >= 0; i-- {
		if err := store.AppendBatch(
			ctx, id.NewStreamRef(cfg.AggType, aggIDs[i]), []event.Event{events[i]},
		); err != nil {
			t.Fatalf("AppendBatch %d: %v", i, err)
		}
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	sorted := make([]event.Event, len(all))
	copy(sorted, all)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].OccurredAt().Equal(sorted[j].OccurredAt()) {
			return sorted[i].ID().String() < sorted[j].ID().String()
		}

		return sorted[i].OccurredAt().Before(sorted[j].OccurredAt())
	})

	for start := 0; start < len(sorted); start += 5 {
		cursor := sorted[start].ID()

		var drained []event.Event

		for {
			batch, err := store.ReadFrom(ctx, cursor, 5)
			if err != nil {
				t.Fatalf("ReadFrom(after %s): %v", cursor, err)
			}

			if len(batch) == 0 {
				break
			}

			drained = append(drained, batch...)
			cursor = batch[len(batch)-1].ID()
		}

		if len(drained) != len(sorted)-start-1 {
			t.Fatalf("drain from %d: got %d events, want %d",
				start, len(drained), len(sorted)-start-1)
		}

		for i := range drained {
			if drained[i].ID() != sorted[start+1+i].ID() {
				t.Fatalf("drain from %d: position %d: got %s, want %s",
					start, i, drained[i].ID(), sorted[start+1+i].ID())
			}
		}
	}
}

func TestPostgresEventStore_ReadFrom_DanglingCursor(t *testing.T) {
	store := pgJournalStore(t)

	cfg := eventtest.IssueStoreConfig()
	aggID := id.NewStreamID()

	evt := cfg.NewTestEvent(t, aggID, 1, event.WithOccurredAt(time.Now()))
	if err := store.AppendBatch(
		context.Background(), id.NewStreamRef(cfg.AggType, aggID), []event.Event{evt},
	); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	batch, err := store.ReadFrom(context.Background(), id.NewEventID(), 100)
	if err != nil {
		t.Fatalf("ReadFrom with dangling cursor: %v", err)
	}

	if len(batch) != 0 {
		t.Fatalf("dangling cursor: got %d events, want 0", len(batch))
	}
}
