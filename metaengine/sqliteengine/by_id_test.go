package sqliteengine

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestStreamLoadByEventID pins the EventByIDBackend capability on the SQLite
// engine (#32): values stored as JSON envelopes with a top-level "id" are
// resolvable by that ID, non-JSON values coexist without breaking lookups,
// and a miss returns metaengine.ErrNotFound.
func TestStreamLoadByEventID(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	defer eng.Close()

	byID, ok := eng.(metaengine.EventByIDBackend)
	if !ok {
		t.Fatal("sqlite engine must implement metaengine.EventByIDBackend")
	}

	ctx := context.Background()
	envelopes := []any{
		`{"id":"e1","type":"order.created","stream_id":"s1"}`,
		`{"id":"e2","type":"order.shipped","stream_id":"s1"}`,
	}
	if err := eng.StreamAppend(ctx, "events", "s1", envelopes); err != nil {
		t.Fatalf("append: %v", err)
	}

	// Non-JSON values in another collection: outside the partial index, and
	// the json_valid guard must keep lookups on OTHER collections working.
	if err := eng.StreamAppend(ctx, "raw", "x", []any{"plain not json"}); err != nil {
		t.Fatalf("raw append: %v", err)
	}

	got, err := byID.StreamLoadByEventID(ctx, "events", "e2")
	if err != nil {
		t.Fatalf("load by event ID: %v", err)
	}

	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("value must decode to a JSON map, got %T", got)
	}
	if m["id"] != "e2" || m["type"] != "order.shipped" {
		t.Fatalf("wrong event resolved: %v", m)
	}

	if _, err := byID.StreamLoadByEventID(ctx, "events", "missing"); !errors.Is(
		err, metaengine.ErrNotFound,
	) {
		t.Fatalf("miss must return metaengine.ErrNotFound, got %v", err)
	}
}
