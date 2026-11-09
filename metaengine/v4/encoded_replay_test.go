package metaengine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

var replayAt = time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)

func newReplayStore(t *testing.T) *metaengine.Store {
	t.Helper()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	return store
}

func cborPayload(t *testing.T, v any) []byte {
	t.Helper()

	payload, err := codec.CBORCodec{}.Encode(v)
	if err != nil {
		t.Fatalf("CBOR encode: %v", err)
	}

	return payload
}

func wantCorruption(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Corruption {
		t.Fatalf("error family = %s, want Corruption (err: %v)", family, err)
	}
}

// TestApplyEncoded_CBOR_ReplayThroughVerify pins the journey of the internal
// CBOR payload marker through EventLog replay: the marker is an unexported type
// crossing the public EventInput.Payload boundary, so a replay regression here
// would silently rebuild projections from nothing.
func TestApplyEncoded_CBOR_ReplayThroughVerify(t *testing.T) {
	t.Parallel()

	store := newReplayStore(t)
	log := metaengine.NewEventLog()
	metaengine.WithEventLog(store, log)

	ctx := context.Background()

	created := cborPayload(t, TaskCreated{
		ID: "t-replay", Title: "Replayed", Assignee: "alice",
		Status: "open", Priority: 2, At: replayAt,
	})
	if err := store.ApplyEncoded(ctx, "TaskCreated", created); err != nil {
		t.Fatalf("ApplyEncoded TaskCreated: %v", err)
	}

	completed := cborPayload(t, TaskCompleted{ID: "t-replay", At: replayAt})
	if err := store.ApplyEncoded(ctx, "TaskCompleted", completed); err != nil {
		t.Fatalf("ApplyEncoded TaskCompleted: %v", err)
	}

	if log.Len() != 2 {
		t.Fatalf("event log length = %d, want 2", log.Len())
	}

	// Verify replays every EventLog entry (CBOR markers included) into a fresh
	// store and compares row counts.
	if err := store.Verify(ctx, []metaengine.Engine{metaengine.NewMemoryEngine()}); err != nil {
		t.Fatalf("Verify after CBOR encoded applies: %v", err)
	}
}

// TestApplyEncoded_CBOR_BackfillReplaysIntoLiveStore pins the same marker on the
// in-place replay path (applyReplay) that Backfill and engine catch-up use.
func TestApplyEncoded_CBOR_BackfillReplaysIntoLiveStore(t *testing.T) {
	t.Parallel()

	store := newReplayStore(t)
	metaengine.WithEventLog(store, metaengine.NewEventLog())

	ctx := context.Background()

	if err := store.ApplyEncoded(ctx, "TaskCreated", cborPayload(t, TaskCreated{
		ID: "t-backfill", Title: "Backfilled", Assignee: "bob",
		Status: "open", Priority: 3, At: replayAt,
	})); err != nil {
		t.Fatalf("ApplyEncoded: %v", err)
	}

	if err := store.Backfill(ctx, metaengine.WithBackfillForce()); err != nil {
		t.Fatalf("Backfill: %v", err)
	}

	result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
		ctx, store, FindTask{ID: "t-backfill"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result.Title != "Backfilled" {
		t.Fatalf("title = %q, want %q", result.Title, "Backfilled")
	}
}

// TestApplyEncodedRecord_SniffsWhenStampUnknown pins the record-level sniff
// path: an unknown Record.Encoding must fall back to detection, not to JSON.
func TestApplyEncodedRecord_SniffsWhenStampUnknown(t *testing.T) {
	t.Parallel()

	store := newReplayStore(t)
	payload := cborPayload(t, TaskCreated{
		ID: "t-sniff", Title: "Sniffed", Assignee: "carol",
		Status: "open", Priority: 1, At: replayAt,
	})

	rec := record.Record{
		Type:     "TaskCreated",
		Payload:  payload,
		StreamID: record.NewStreamRef("Task", "t-sniff"),
		Version:  1,
	}

	if err := store.ApplyEncodedRecord(context.Background(), rec, payload); err != nil {
		t.Fatalf("ApplyEncodedRecord with unknown stamp: %v", err)
	}

	result, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
		context.Background(), store, FindTask{ID: "t-sniff"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result.Title != "Sniffed" {
		t.Fatalf("title = %q, want %q", result.Title, "Sniffed")
	}
}

// TestApplyEncodedRecord_TrustsKnownStamp pins the other half of the contract:
// a stamp that lies about the bytes surfaces as a deterministic Corruption
// error instead of being silently re-sniffed.
func TestApplyEncodedRecord_TrustsKnownStamp(t *testing.T) {
	t.Parallel()

	store := newReplayStore(t)
	payload := cborPayload(t, TaskCreated{ID: "t-lie", Title: "Liar", At: replayAt})

	rec := record.Record{
		Type:     "TaskCreated",
		Encoding: record.EncodingJSON, // lie: bytes are CBOR
		Payload:  payload,
		StreamID: record.NewStreamRef("Task", "t-lie"),
		Version:  1,
	}

	err := store.ApplyEncodedRecord(context.Background(), rec, payload)
	wantCorruption(t, err)

	if !strings.Contains(err.Error(), "decode TaskCreated") {
		t.Fatalf("error %q does not name the event type", err)
	}
}

// TestApplyEncoded_RejectsEmptyPayload pins the ingress guard.
func TestApplyEncoded_RejectsEmptyPayload(t *testing.T) {
	t.Parallel()

	store := newReplayStore(t)

	err := store.ApplyEncoded(context.Background(), "TaskCreated", nil)
	wantCorruption(t, err)

	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error %q does not explain the empty payload", err)
	}
}

// TestApplyEncoded_ByteSampleFoldReceivesRawCBOR pins the isByteSample branch:
// a fold whose sample is a byte slice consumes the producer's bytes undecoded.
func TestApplyEncoded_ByteSampleFoldReceivesRawCBOR(t *testing.T) {
	t.Parallel()

	var received []byte

	query := metaengine.Query[string, int](
		"raw-bytes",
		metaengine.OnRecordTyped(
			"raw.payload",
			[]byte(nil),
			func(_ record.Record, e []byte) (string, int) {
				received = append([]byte(nil), e...)
				return "raw", len(e)
			},
		),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		query,
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	payload := cborPayload(t, TaskCreated{ID: "raw", Title: "Raw", At: replayAt})

	if err := store.ApplyEncoded(context.Background(), "raw.payload", payload); err != nil {
		t.Fatalf("ApplyEncoded: %v", err)
	}

	if len(received) == 0 {
		t.Fatal("byte-sample fold received no bytes")
	}

	if string(received) != string(payload) {
		t.Fatal("byte-sample fold did not receive the raw CBOR bytes")
	}
}

// TestApplyEncoded_GarbageDoesNotSucceed pins that unclassifiable bytes never
// reach a fold as a zero value.
func TestApplyEncoded_GarbageDoesNotSucceed(t *testing.T) {
	t.Parallel()

	store := newReplayStore(t)

	err := store.ApplyEncoded(
		context.Background(), "TaskCreated", []byte{0x1c, 0x1c, 0x1c},
	)
	wantCorruption(t, err)
}

// TestApplyEncoded_DoesNotSwallowApplyErrors keeps the existing contract: an
// event type with no matching fold is not an error, and a payload that cannot
// decode into the fold's sample is.
func TestApplyEncoded_DoesNotSwallowApplyErrors(t *testing.T) {
	t.Parallel()

	store := newReplayStore(t)

	if err := store.ApplyEncoded(
		context.Background(), "UnrelatedEvent", cborPayload(t, TaskCreated{ID: "x", At: replayAt}),
	); err != nil {
		t.Fatalf("unmatched event type should be a no-op, got %v", err)
	}

	wantCorruption(t,
		store.ApplyEncoded(context.Background(), "TaskCreated", cborPayload(t, "not a task")),
	)
}
