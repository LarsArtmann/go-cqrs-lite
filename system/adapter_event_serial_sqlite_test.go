package system_test

import (
	"bytes"
	"context"
	"database/sql"
	"maps"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"
	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type cborTaskPayload struct {
	Title string    `json:"title"`
	Due   time.Time `json:"due"`
	Tags  []string  `json:"tags"`
}

// TestEventAdapter_CBORPayloadRoundTripSQLite pins the persistence contract
// that DecodePayloadAuto consumers rely on: a CBOR-encoded payload survives
// the SQL envelope path (encodeEvent to TEXT, decodeEvent back) byte-identical,
// with the encoding tag and metadata intact. A pre-system journal migration
// (go-cqrs-lite#58) only needs to reproduce this envelope, not understand the
// payload.
func TestEventAdapter_CBORPayloadRoundTripSQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "journal.db"))
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

	adapter := system.NewEventAdapter(backend, "events", system.WithSerialization())

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Task", streamID)

	wantPayload := cborTaskPayload{
		Title: "quarterly review",
		Due:   time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC),
		Tags:  []string{"cbor", "sqlite"},
	}
	wantMeta := event.Metadata{
		Custom: map[event.MetadataKey]string{"tenant": "acme", "source": "test"},
	}

	evt, err := event.New(
		"task.created",
		streamID,
		"Task",
		1,
		wantPayload,
		event.WithMetadata(wantMeta),
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if got := evt.Encoding(); got != codec.EncodingCBOR {
		t.Fatalf("event.New encoding = %v, want CBOR (v4 default codec)", got)
	}

	if err := adapter.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := adapter.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Load returned %d events, want 1", len(loaded))
	}

	got := loaded[0]

	if got.ID() != evt.ID() {
		t.Errorf("event ID mismatch: got %s, want %s", got.ID(), evt.ID())
	}

	if got.Type() != evt.Type() {
		t.Errorf("event type mismatch: got %s, want %s", got.Type(), evt.Type())
	}

	if got.Version() != evt.Version() {
		t.Errorf("version mismatch: got %v, want %v", got.Version(), evt.Version())
	}

	if !bytes.Equal(event.PayloadReadOnly(got), event.PayloadReadOnly(evt)) {
		t.Errorf("payload bytes not preserved through the envelope:\n got: %x\nwant: %x",
			event.PayloadReadOnly(got), event.PayloadReadOnly(evt))
	}

	if got.Encoding() != codec.EncodingCBOR {
		t.Errorf("encoding tag mismatch: got %v, want %v", got.Encoding(), codec.EncodingCBOR)
	}

	if !maps.Equal(got.Metadata().Custom, wantMeta.Custom) {
		t.Errorf(
			"custom metadata mismatch: got %v, want %v",
			got.Metadata().Custom,
			wantMeta.Custom,
		)
	}

	if !got.OccurredAt().Equal(evt.OccurredAt()) {
		t.Errorf("occurred-at mismatch: got %v, want %v", got.OccurredAt(), evt.OccurredAt())
	}

	decoded, err := event.DecodePayloadAuto[cborTaskPayload](got)
	if err != nil {
		t.Fatalf("DecodePayloadAuto: %v", err)
	}

	// The CBOR codec normalizes time.Location on decode (same instant, local
	// offset), so compare fields with instant equality for times. The envelope
	// itself is byte-faithful — asserted above via PayloadReadOnly.
	if decoded.Title != wantPayload.Title ||
		!decoded.Due.Equal(wantPayload.Due) ||
		!slices.Equal(decoded.Tags, wantPayload.Tags) {
		t.Errorf("decoded payload mismatch: got %+v, want %+v", decoded, wantPayload)
	}
}
