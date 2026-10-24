package adttest

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// walEnvelope builds a wire envelope the way the system adapter does —
// json.Marshal of the real shape — so the []byte fields carry guaranteed-
// valid base64 (hand-written fixture strings once shipped INVALID base64 and
// the json decode of the payload field failed, silently disabling typed
// extraction for the whole row: the A11 red-pin lesson).
func walEnvelope(t *testing.T, eventID, streamID string, version int) string {
	t.Helper()

	data, err := json.Marshal(map[string]any{
		"id":             eventID,
		"type":           "t.evt",
		"stream_id":      streamID,
		"stream_type":    "Task",
		"version":        version,
		"schema_version": 0,
		"payload":        []byte{0xa1, 0x02, 0x69, 0x64},
		"encoding":       "cbor",
		"metadata":       []byte(`{}`),
		"occurred_at":    "2026-10-23T15:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	return string(data)
}

// AssertWALDuplicateAppendConflicts pins the journal-of-record identity
// contract (disk-truth A14): appending the SAME event envelope (same event
// id) twice to a stream must surface metaengine.ErrVersionConflict — the
// optimistic-concurrency contract — instead of silently adding a second
// journal row. Run it from every engine module's test suite; the scenario is
// engine-agnostic, only the engine setup differs per module.
func AssertWALDuplicateAppendConflicts(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatalf("engine does not implement metaengine.StreamLogBackend")
	}

	// Run-unique collection: a fixed name lets a previous run's rows make the
	// next run's FIRST append conflict on persistent database surfaces (the
	// adttest rerun-safety rule; same as tx_iso/dedup/temporal conformance).
	col := fmt.Sprintf("wal_dup_%d", time.Now().UnixNano())

	ctx := context.Background()
	env := walEnvelope(t, "01WALDUP0000000000000000001", "01WALSTREAM0000000000001", 1)

	if err := backend.StreamAppend(ctx, col, "Task:01WALSTREAM0000000000001", []any{env}); err != nil {
		t.Fatalf("first append: %v", err)
	}

	err := backend.StreamAppend(ctx, col, "Task:01WALSTREAM0000000000001", []any{env})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("duplicate append error = %v, want metaengine.ErrVersionConflict", err)
	}
}

// AssertWALStreamVersionConflict pins the second identity constraint
// (disk-truth A14): appending a DIFFERENT event into an occupied
// (collection, stream, version) slot must surface
// metaengine.ErrVersionConflict — the journal is a versioned stream, not a
// bag of rows. Run it from every engine module's test suite.
func AssertWALStreamVersionConflict(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	backend, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatalf("engine does not implement metaengine.StreamLogBackend")
	}

	col := fmt.Sprintf("wal_ver_%d", time.Now().UnixNano())

	ctx := context.Background()

	if err := backend.StreamAppend(ctx, col, "Task:01WALSTREAM0000000000001",
		[]any{walEnvelope(t, "01WALVER00000000000000000001", "01WALSTREAM0000000000001", 1)}); err != nil {
		t.Fatalf("seed append: %v", err)
	}

	// Same (collection, stream, version), DIFFERENT event id: the slot is
	// occupied and the append must conflict.
	err := backend.StreamAppend(ctx, col, "Task:01WALSTREAM0000000000001",
		[]any{walEnvelope(t, "01WALVER00000000000000000002", "01WALSTREAM0000000000001", 1)})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("(stream,version) duplicate error = %v, want metaengine.ErrVersionConflict", err)
	}
}
