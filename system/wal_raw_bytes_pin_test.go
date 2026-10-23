package system_test

// Raw-bytes WAL pins (disk-truth plan A6, 2026-10-23): round-trip tests
// cannot see WIRE-FORM regressions — a writer that swaps the bare stream ID
// for the branded display form, double-encodes the payload, or breaks the
// metadata JSON round-trips green while persisting bytes no other tool can
// read (proven twice before this pin existed). These tests read the RAW
// meta_stream_log rows a real SQLite engine persisted and assert the
// envelope contract directly.

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json/v2"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type pinTaskPayload struct {
	Title string `json:"title"`
	N     int    `json:"n"`
}

const marker = "StreamMarker:"

func openPinEngine(t *testing.T) (metaengine.StreamLogBackend, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "pin.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("sqlite engine must implement metaengine.StreamLogBackend")
	}

	return backend, db
}

// rawWALRow reads the persisted journal row for a stream key.
func rawWALRow(t *testing.T, db *sql.DB, collection, bareKey string) (streamID, value string) {
	t.Helper()

	err := db.QueryRow(
		`SELECT stream_id, CAST(value AS TEXT) FROM meta_stream_log
		 WHERE collection = ? AND stream_id = ? LIMIT 1`,
		collection, bareKey,
	).Scan(&streamID, &value)
	if err != nil {
		t.Fatalf("read raw WAL row (%s/%s): %v", collection, bareKey, err)
	}

	return streamID, value
}

// TestWALPin_EventEnvelopeBareAndRaw asserts the EVENT WAL wire form: the
// stream_id COLUMN and envelope field are the bare wire value (no
// "StreamMarker:" display form), the payload rides as plain base64-of-raw
// bytes (no double encoding) with its encoding stamp, the metadata is a
// JSON object, and occurred_at is RFC3339 UTC with nanosecond precision.
func TestWALPin_EventEnvelopeBareAndRaw(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	backend, db := openPinEngine(t)
	adapter := system.NewEventAdapter(backend, "pin_events", system.WithSerialization())

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Task", streamID)
	payload := pinTaskPayload{Title: "raw-pin", N: 42}

	evt, err := event.New("pin.evt", streamID, "Task", 1, payload)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if err := adapter.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	bareKey := "Task:" + streamID.Get()
	streamIDCol, value := rawWALRow(t, db, "pin_events", bareKey)

	if streamIDCol != bareKey {
		t.Fatalf("stream_id column = %q, want bare %q (no display form)", streamIDCol, bareKey)
	}

	if !strings.HasPrefix(value, "{") {
		t.Fatalf("value column is not a JSON object: %.60q", value)
	}

	var env struct {
		StreamID   string `json:"stream_id"`
		Payload    string `json:"payload"`
		Encoding   string `json:"encoding"`
		Metadata   string `json:"metadata"`
		OccurredAt string `json:"occurred_at"`
	}

	if err := json.Unmarshal([]byte(value), &env); err != nil {
		t.Fatalf("envelope is not valid JSON: %v", err)
	}

	if env.StreamID != streamID.Get() {
		t.Fatalf("envelope stream_id = %q, want bare %q", env.StreamID, streamID.Get())
	}

	if strings.Contains(env.StreamID, marker) {
		t.Fatalf("envelope stream_id carries the %q display form", marker)
	}

	rawPayload, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		t.Fatalf("payload is not plain base64 of raw bytes (double-encoding?): %v", err)
	}

	if !equalBytes(rawPayload, event.PayloadReadOnly(evt)) {
		t.Fatalf("payload bytes differ from the event's raw payload:\n got: %x\nwant: %x",
			rawPayload, event.PayloadReadOnly(evt))
	}

	if env.Encoding != "cbor" {
		t.Fatalf("encoding stamp = %q, want cbor (v4 default)", env.Encoding)
	}

	rawMeta, err := base64.StdEncoding.DecodeString(env.Metadata)
	if err != nil {
		t.Fatalf("metadata is not plain base64 of JSON bytes: %v", err)
	}

	var metaObj map[string]any
	if err := json.Unmarshal(rawMeta, &metaObj); err != nil {
		t.Fatalf("metadata is not a JSON object: %v (raw: %.60q)", err, rawMeta)
	}

	if !strings.HasSuffix(env.OccurredAt, "Z") {
		t.Fatalf("occurred_at %q is not UTC (Z-suffix)", env.OccurredAt)
	}

	if _, err := time.Parse(time.RFC3339Nano, env.OccurredAt); err != nil {
		t.Fatalf("occurred_at %q is not RFC3339Nano: %v", env.OccurredAt, err)
	}
}

// TestWALPin_CommandEnvelopeBareAndRaw is the command-WAL twin of the event
// pin: same invariants over serializedCommand rows.
func TestWALPin_CommandEnvelopeBareAndRaw(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	backend, db := openPinEngine(t)
	adapter := system.NewCommandAdapter(backend, "pin_commands", system.WithCommandSerialization())

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Task", streamID)

	pc, err := command.NewPersistedCommand(
		"pin.cmd", ref, []byte(`{"title":"cmd-pin","n":7}`),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := adapter.Save(ctx, ref, pc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	bareKey := "Task:" + streamID.Get()
	streamIDCol, value := rawWALRow(t, db, "pin_commands", bareKey)

	if streamIDCol != bareKey {
		t.Fatalf("stream_id column = %q, want bare %q", streamIDCol, bareKey)
	}

	var env struct {
		StreamID   string `json:"stream_id"`
		Payload    string `json:"payload"`
		Metadata   string `json:"metadata"`
		ReceivedAt string `json:"received_at"`
	}

	if err := json.Unmarshal([]byte(value), &env); err != nil {
		t.Fatalf("envelope is not valid JSON: %v", err)
	}

	if env.StreamID != streamID.Get() || strings.Contains(env.StreamID, marker) {
		t.Fatalf("envelope stream_id = %q, want bare %q", env.StreamID, streamID.Get())
	}

	rawPayload, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		t.Fatalf("payload is not plain base64 of raw bytes: %v", err)
	}

	if !equalBytes(rawPayload, pc.Payload()) {
		t.Fatalf("payload bytes differ from the command's raw payload")
	}

	rawMeta, err := base64.StdEncoding.DecodeString(env.Metadata)
	if err != nil {
		t.Fatalf("metadata is not plain base64 of JSON bytes: %v", err)
	}

	var metaObj map[string]any
	if err := json.Unmarshal(rawMeta, &metaObj); err != nil {
		t.Fatalf("metadata is not a JSON object: %v", err)
	}

	if !strings.HasSuffix(env.ReceivedAt, "Z") {
		t.Fatalf("received_at %q is not UTC (Z-suffix)", env.ReceivedAt)
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
