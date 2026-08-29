package storage

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// seedTiedEvents appends n events where each group of 3 shares the exact same
// occurred_at, forcing the (occurred_at, id) tie-break in journal ordering.
// Events are inserted newest-first so rowid order diverges from journal order —
// an ORDER BY that silently relies on insertion order fails this seeding.
func seedTiedEvents(t *testing.T, store *SQLEventStore, n int) []event.Event {
	t.Helper()

	cfg := issueStoreConfig()
	base := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()

	events := make([]event.Event, n)
	aggIDs := make([]id.StreamID, n)

	for i := 0; i < n; i++ {
		aggIDs[i] = id.NewStreamID()
		occurred := base.Add(time.Duration(i/3) * time.Millisecond)

		events[i] = cfg.NewTestEvent(t, aggIDs[i], 1, event.WithOccurredAt(occurred))
	}

	for i := n - 1; i >= 0; i-- {
		if err := store.AppendBatch(
			ctx,
			id.NewStreamRef(cfg.AggType, aggIDs[i]),
			[]event.Event{events[i]},
		); err != nil {
			t.Fatalf("AppendBatch %d: %v", i, err)
		}
	}

	return events
}

// canonicalJournalOrder sorts events the way the journal pagination orders
// them: occurred_at ASC, then id ASC. ReadAll only orders by occurred_at, so
// its tie order is unspecified — comparisons must go through this function.
func canonicalJournalOrder(events []event.Event) []event.Event {
	sorted := make([]event.Event, len(events))
	copy(sorted, events)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].OccurredAt().Equal(sorted[j].OccurredAt()) {
			return sorted[i].ID().String() < sorted[j].ID().String()
		}

		return sorted[i].OccurredAt().Before(sorted[j].OccurredAt())
	})

	return sorted
}

func assertEventIDs(t *testing.T, got, want []event.Event, contextMsg string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: got %d events, want %d", contextMsg, len(got), len(want))
	}

	for i := range want {
		if got[i].ID() != want[i].ID() {
			t.Fatalf("%s: position %d: got event %s, want %s",
				contextMsg, i, got[i].ID(), want[i].ID())
		}
	}
}

func TestSQLiteEventStore_ReadFrom_KeysetEquivalence(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	events := seedTiedEvents(t, store, 57) // 19 tie-groups of 3
	ctx := context.Background()

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	ordered := canonicalJournalOrder(all)
	assertEventIDs(t, ordered, canonicalJournalOrder(events), "canonical order vs seeded events")

	// Drain from every 7th event (crossing tie-group boundaries) in batches
	// of 5; the concatenated drain must equal the ordered suffix after the
	// cursor, with no duplicates and no gaps.
	for start := 0; start < len(ordered); start += 7 {
		cursor := ordered[start].ID()

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

		assertEventIDs(
			t,
			drained,
			ordered[start+1:],
			"drain from position "+ordered[start].ID().String(),
		)
	}
}

func TestSQLiteEventStore_ReadFrom_DanglingCursor(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	seedTiedEvents(t, store, 6)

	missing := id.NewEventID()

	batch, err := store.ReadFrom(context.Background(), missing, 100)
	if err != nil {
		t.Fatalf("ReadFrom with dangling cursor: %v", err)
	}

	if len(batch) != 0 {
		t.Fatalf("dangling cursor: got %d events, want 0", len(batch))
	}
}

func TestSQLiteEventStore_ReadStreamFrom_KeysetEquivalence(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	events := seedTiedEvents(t, store, 30)
	ctx := context.Background()

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	ordered := canonicalJournalOrder(all)
	assertEventIDs(t, ordered, canonicalJournalOrder(events), "canonical order vs seeded events")

	cursor := ordered[4].ID()

	var drained []event.Event

	for {
		iter, err := store.ReadStreamFrom(ctx, cursor, 7)
		if err != nil {
			t.Fatalf("ReadStreamFrom(after %s): %v", cursor, err)
		}

		var batchCount int

		for {
			evt, err := iter.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("iterate: %v", err)
			}

			drained = append(drained, evt)
			batchCount++
		}

		if err := iter.Close(); err != nil {
			t.Fatalf("close iterator: %v", err)
		}

		if batchCount == 0 {
			break
		}

		cursor = drained[len(drained)-1].ID()
	}

	assertEventIDs(t, drained, ordered[5:], "stream drain")
}

func TestSQLiteEventStore_ReadStreamFrom_DanglingCursor(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)
	seedTiedEvents(t, store, 3)

	iter, err := store.ReadStreamFrom(context.Background(), id.NewEventID(), 10)
	if err != nil {
		t.Fatalf("ReadStreamFrom with dangling cursor: %v", err)
	}
	defer func() { _ = iter.Close() }()

	if _, err := iter.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("dangling cursor: Next() = %v, want io.EOF", err)
	}
}

// TestSQLiteJournalReadFrom_UsesTimestampIndex pins the query plan of the
// keyset pagination: it must be driven by idx_events_occurred_at and must not
// fall back to a MULTI-INDEX OR plan or a full temp B-tree sort of the result
// (the failure mode of the former self-JOIN cursor query, which made batched
// journal drains O(N²)).
func TestSQLiteJournalReadFrom_UsesTimestampIndex(t *testing.T) {
	t.Parallel()

	db := newSQLiteTestDB(t)
	initSQLiteSchema(t, db)

	if _, err := db.ExecContext(
		context.Background(),
		`INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, occurred_at)
		 VALUES ('cursor-id', 'TestEvent', 'Issue', 'agg', 1, '2026-08-16T00:00:00.000000001Z')`,
	); err != nil {
		t.Fatalf("seed cursor row: %v", err)
	}

	query, err := sqlpkg.KeysetPositionQueryChecked(
		sqlpkg.SQLiteDialect{}, "e.id, e.occurred_at", sqlpkg.TableEvents, "occurred_at",
	)
	if err != nil {
		t.Fatalf("KeysetPositionQueryChecked: %v", err)
	}

	rows, err := db.QueryContext(
		context.Background(),
		"EXPLAIN QUERY PLAN "+query,
		"2026-08-16T00:00:00.000000001Z", "2026-08-16T00:00:00.000000001Z", "cursor-id",
	)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN: %v", err)
	}
	defer sqlpkg.CloseRows(rows)

	var plan strings.Builder

	for rows.Next() {
		var id, parent, notused, detail any

		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan plan row: %v", err)
		}

		if s, ok := detail.(string); ok {
			plan.WriteString(s)
			plan.WriteString("\n")
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate plan: %v", err)
	}

	details := plan.String()
	if !strings.Contains(details, "USING INDEX idx_events_occurred_at") {
		t.Errorf("plan does not use idx_events_occurred_at:\n%s", details)
	}

	if strings.Contains(details, "MULTI-INDEX OR") ||
		strings.Contains(details, "TEMP B-TREE FOR ORDER BY") {
		t.Errorf("plan regressed to multi-index OR / full temp sort:\n%s", details)
	}
}

// BenchmarkSQLiteEventStore_ReadFrom_FullDrain measures a full journal drain
// in projectionhost-sized batches (100). Regression guard for the O(N²)
// self-JOIN cursor pagination: draining 5k events must stay in the
// milliseconds, not seconds.
func BenchmarkSQLiteEventStore_ReadFrom_FullDrain(b *testing.B) {
	db, err := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	for _, ddl := range []string{sqlpkg.SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema()} {
		if _, err := db.ExecContext(context.Background(), ddl); err != nil {
			b.Fatalf("exec DDL: %v\nDDL: %s", err, ddl)
		}
	}

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		b.Fatalf("NewSQLiteEventStore: %v", err)
	}

	ctx := context.Background()

	const total = 5000
	cfg := eventtest.IssueStoreConfig()
	base := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	aggID := id.NewStreamID()

	batch := make([]event.Event, 0, total)

	for i := 0; i < total; i++ {
		evt, err := event.NewEvent(
			cfg.EvtType,
			aggID,
			cfg.AggType,
			event.Version(i+1),
			cfg.Payload(event.Version(i+1)),
			event.WithOccurredAt(base.Add(time.Duration(i)*time.Microsecond)),
		)
		if err != nil {
			b.Fatalf("create event %d: %v", i, err)
		}

		batch = append(batch, evt)
	}

	for i := 0; i < total; i += 500 {
		end := min(i+500, total)

		if err := store.AppendBatch(
			ctx,
			id.NewStreamRef(cfg.AggType, aggID),
			batch[i:end],
		); err != nil {
			b.Fatalf("AppendBatch: %v", err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cursor := id.EventID{}

		var count int

		for {
			events, err := store.ReadFrom(ctx, cursor, 100)
			if err != nil {
				b.Fatalf("ReadFrom: %v", err)
			}

			if len(events) == 0 {
				break
			}

			count += len(events)
			cursor = events[len(events)-1].ID()
		}

		if count != total {
			b.Fatalf("drained %d events, want %d", count, total)
		}
	}
}
