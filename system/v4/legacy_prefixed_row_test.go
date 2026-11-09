package system_test

// Legacy prefixed-row fixture (disk-truth plan A23): a journal row written
// by a PRE-bare-key deployment (stream_id keyed by the branded DISPLAY form,
// "TaskMarker:<raw>") stays readable through the CURRENT adapter — sticky
// resolution finds it, loads return it, and new appends continue under the
// SAME legacy key so the stream's history never splits.

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// legacyEnvelope builds a v4-era event envelope with json.Marshal of the
// real adapter shape (the wire-fixture rule: generated through the producing
// serializer, never hand-written base64).
func legacyEnvelope(t *testing.T, eventID, streamID string, version int) string {
	t.Helper()

	data, err := json.Marshal(map[string]any{
		"id":          eventID,
		"type":        "task.completed",
		"stream_id":   streamID,
		"stream_type": "Task",
		"version":     version,
		"payload":     []byte{0xa1, 0x02, 0x69, 0x64},
		"encoding":    "cbor",
		"metadata":    []byte(`{}`),
		"occurred_at": "2026-10-17T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal legacy envelope: %v", err)
	}

	return string(data)
}

func TestLegacyPrefixedRows_StickyKeyReadAndAppend(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "legacy-keys.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	backend := eng.(metaengine.StreamLogBackend)

	streamID, err := id.ParseStreamID("01LEGACY0000000000000001")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	ref := id.NewStreamRef("Task", streamID)

	// The v4-era storage key: the branded DISPLAY form, prefix and all.
	legacyKey := ref.Type.String() + ":" + ref.ID.String()
	if legacyKey == "Task:"+streamID.Get() {
		t.Fatalf("fixture premise broken: the display form equals the bare form (%s)", legacyKey)
	}

	const collection = "events"

	if err := backend.StreamAppend(
		context.Background(), collection, legacyKey,
		[]any{legacyEnvelope(t, "01LEGEVENT0000000000000001", streamID.Get(), 1)},
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// WithSerialization: a real SQLite-backed deployment persists event
	// ENVELOPES (the adapter's JSON wrapper), not raw *event.ImmutableEvent
	// pointers — the engine's encode gate (2026-10-23) rightly rejects
	// structs it cannot marshal, where the old path silently wrote a
	// fmt.Sprintf placeholder this test's row counting never noticed.
	adapter := system.NewEventAdapter(backend, collection, system.WithSerialization())

	// Load through the CURRENT adapter: sticky resolution must find the
	// legacy-keyed row.
	loaded, err := adapter.Load(context.Background(), ref)
	if err != nil {
		t.Fatalf("load legacy-keyed stream: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf(
			"legacy stream length = %d, want 1 (sticky resolution missed the row)",
			len(loaded),
		)
	}

	// Append through the CURRENT adapter: the write must CONTINUE under the
	// legacy key (sticky), never split the history under the bare key.
	next, err := event.New("task.completed", streamID, "Task", 2, []byte{0xa1, 0x02, 0x69, 0x64})
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	if err := adapter.Save(context.Background(), ref, []event.Event{next}, 1); err != nil {
		t.Fatalf("save onto legacy-keyed stream: %v", err)
	}

	var legacyRows, bareRows int

	if err := db.QueryRow(`SELECT COUNT(*) FROM meta_stream_log
		WHERE collection = ? AND stream_id = ?`, collection, legacyKey).Scan(&legacyRows); err != nil {
		t.Fatalf("count legacy-keyed rows: %v", err)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM meta_stream_log
		WHERE collection = ? AND stream_id = ?`, collection, "Task:"+streamID.Get()).Scan(&bareRows); err != nil {
		t.Fatalf("count bare-keyed rows: %v", err)
	}

	if legacyRows != 2 {
		t.Fatalf("stream must stay under its legacy key (2 rows), got %d", legacyRows)
	}

	if bareRows != 0 {
		t.Fatalf("append split the stream history: %d bare-keyed rows appeared", bareRows)
	}
}
