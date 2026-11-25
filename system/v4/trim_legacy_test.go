package system_test

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_TrimStream_LegacyKeyedStreamAlsoErasesBareSnapshot pins the
// sticky-key hole in the GDPR-erasure primitive: a stream whose journal rows
// live under the pre-2026-10-17 legacy key is trimmed under that key, while
// its snapshot is always written under the bare key. TrimStream must erase
// both, or Load still finds the snapshot of an erased stream.
func TestSystem_TrimStream_LegacyKeyedStreamAlsoErasesBareSnapshot(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := "file:" + filepath.Join(t.TempDir(), "trim-legacy.db")

	streamID, err := id.ParseStreamID("01LEGACY0000000000000002")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	ref := id.NewStreamRef("Task", streamID)
	legacyKey := ref.Type.String() + ":" + ref.ID.String()

	if legacyKey == ref.StreamKey() {
		t.Fatalf("fixture premise broken: display form equals bare form (%s)", legacyKey)
	}

	seedLegacyJournalRow(t, dsn, legacyKey, streamID.Get())

	sys, err := system.New(ctx, system.DomainConfig{}, trimDeployment(dsn))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer func() { _ = sys.Close() }()

	if sys.SnapshotStore() == nil {
		t.Fatal("fixture premise broken: the deployment exposes no snapshot store")
	}

	if err := sys.SnapshotStore().Save(ctx, snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "Task",
		Version:    1,
		State:      []byte(`{"state":"legacy"}`),
	}); err != nil {
		t.Fatalf("snapshot save: %v", err)
	}

	if events, err := sys.EventStore().Load(ctx, ref); err != nil || len(events) != 1 {
		t.Fatalf("premise: legacy stream load = (%d events, %v), want (1, <nil>)", len(events), err)
	}

	removed, err := sys.TrimStream(ctx, ref)
	if err != nil {
		t.Fatalf("TrimStream: %v", err)
	}

	if removed != 1 {
		t.Fatalf("TrimStream removed %d rows, want 1", removed)
	}

	if got, err := sys.SnapshotStore().Load(ctx, ref); err != nil || got != nil {
		t.Fatalf("snapshot of the erased legacy-keyed stream survived: (%v, %v), want (nil, <nil>)", got, err)
	}

	if events, err := sys.EventStore().Load(ctx, ref); err != nil || len(events) != 0 {
		t.Fatalf("Load after trim: (%d events, %v), want (0, <nil>)", len(events), err)
	}
}

// seedLegacyJournalRow writes one journal row the way a pre-bare-key
// deployment did: keyed by the branded display form, through the engine's
// own append so the envelope is the real wire shape.
func seedLegacyJournalRow(t *testing.T, dsn, legacyKey, streamID string) {
	t.Helper()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	envelope, err := json.Marshal(map[string]any{
		"id":          "01LEGEVENT0000000000000002",
		"type":        "task.completed",
		"stream_id":   streamID,
		"stream_type": "Task",
		"version":     1,
		"payload":     []byte{0xa1, 0x02, 0x69, 0x64},
		"encoding":    "cbor",
		"metadata":    []byte(`{}`),
		"occurred_at": "2026-10-17T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal legacy envelope: %v", err)
	}

	const collection = "events"

	if err := eng.(metaengine.StreamLogBackend).StreamAppend(
		context.Background(), collection, legacyKey, []any{string(envelope)},
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	if err := eng.Close(); err != nil {
		t.Fatalf("close seeding engine: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close seeding db: %v", err)
	}
}
