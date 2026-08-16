package sql_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// unknownDialect is a custom Dialect that deliberately does not embed any
// known dialect, so MaxParametersForDialect must treat it conservatively.
type unknownDialect struct{ sqlpkg.PostgresDialect }

func sqliteFormatTime(t time.Time) any { return t.Format(time.RFC3339Nano) }

func TestMaxParametersForDialect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dialect sqlpkg.Dialect
		want    int
	}{
		{"sqlite", sqlpkg.SQLiteDialect{}, 999},
		{"postgres", sqlpkg.PostgresDialect{}, 32767},
		{"mysql", sqlpkg.MySQLDialect{}, 32767},
		{"duckdb", sqlpkg.DuckDBDialect{}, 32767},
		{"unknown custom keeps sqlite-safe limit", unknownDialect{}, 999},
		{"nil keeps sqlite-safe limit", nil, 999},
	}

	for _, tt := range tests {
		if got := sqlpkg.MaxParametersForDialect(tt.dialect); got != tt.want {
			t.Errorf("%s: MaxParametersForDialect = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestRowsWithinByteCap(t *testing.T) {
	t.Parallel()

	const mib = 1 << 20
	tests := []struct {
		name    string
		sizes   []int
		start   int
		maxRows int
		want    int
	}{
		{"param cap binds for small rows", filled(10, 1024), 0, 5, 5},
		{"remainder shorter than maxRows", filled(7, 1024), 5, 5, 2},
		{
			"byte cap splits at boundary (total == cap stays)",
			filled(10, 4*mib), 0, 10, 2,
		},
		{"byte cap splits mid-batch", filled(10, 5*mib), 0, 10, 1},
		{"single oversized row still returns one", []int{64 * mib}, 0, 10, 1},
		{"zero maxRows clamps to one row", filled(3, 1024), 0, 0, 1},
		{"start at end returns zero", filled(3, 1024), 3, 5, 0},
	}

	for _, tt := range tests {
		got := sqlpkg.RowsWithinByteCap(tt.start, len(tt.sizes), tt.maxRows, func(i int) int {
			return tt.sizes[i]
		})
		if got != tt.want {
			t.Errorf("%s: RowsWithinByteCap = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func filled(n, size int) []int {
	sizes := make([]int, n)
	for i := range sizes {
		sizes[i] = size
	}

	return sizes
}

func makeLargeTestEvent(t *testing.T, version, payloadBytes int) event.Event {
	t.Helper()

	payload := make([]byte, payloadBytes)
	for i := range payload {
		payload[i] = byte('a' + i%26)
	}

	evt, err := event.NewEvent(
		event.Type("document.attached"),
		id.NewStreamID(),
		id.StreamType("User"),
		event.Version(version),
		payload,
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

// TestSharedBatchInsertEvents_LargePayloadsChunkSafely proves the byte cap
// keeps multi-VALUES statements under MaxStatementBytes: 9 one-MiB payloads
// cannot share one statement (9 MiB > 8 MiB cap), so the loop must split while
// still inserting every event exactly once.
func TestSharedBatchInsertEvents_LargePayloadsChunkSafely(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := id.NewStreamRef("User", id.NewStreamID())
	ctx := context.Background()

	const eventCount = 9
	events := make([]event.Event, eventCount)
	for i := range events {
		events[i] = makeLargeTestEvent(t, i+1, 1<<20)
	}

	tx := beginTx(t, db)
	defer rollbackOnFail(t, tx)

	if err := sqlpkg.SharedBatchInsertEvents(
		ctx, tx, ref, events, sqlpkg.SQLiteDialect{}, sqliteFormatTime,
	); err != nil {
		t.Fatalf("SharedBatchInsertEvents: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != eventCount {
		t.Errorf("count = %d, want %d", count, eventCount)
	}

	var payload []byte
	query := fmt.Sprintf("SELECT payload FROM %s WHERE version = ?", sqlpkg.TableEvents)
	if err := db.QueryRowContext(ctx, query, eventCount).Scan(&payload); err != nil {
		t.Fatalf("load payload: %v", err)
	}
	if len(payload) != 1<<20 {
		t.Errorf("payload length = %d, want %d", len(payload), 1<<20)
	}
}

// TestSharedBatchInsertEvents_ManySmallEvents proves the param cap still
// chunks correctly (SQLite: 99 rows per statement) and the loop terminates.
func TestSharedBatchInsertEvents_ManySmallEvents(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := id.NewStreamRef("User", id.NewStreamID())
	ctx := context.Background()

	const eventCount = 250
	events := make([]event.Event, eventCount)
	for i := range events {
		events[i] = makeTestEvent(t, i+1)
	}

	tx := beginTx(t, db)
	defer rollbackOnFail(t, tx)

	if err := sqlpkg.SharedBatchInsertEvents(
		ctx, tx, ref, events, sqlpkg.SQLiteDialect{}, sqliteFormatTime,
	); err != nil {
		t.Fatalf("SharedBatchInsertEvents: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != eventCount {
		t.Errorf("count = %d, want %d", count, eventCount)
	}
}
