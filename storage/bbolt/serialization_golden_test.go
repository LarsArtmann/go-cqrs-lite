package bbolt

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// The golden test pins the serialized wire format of serializableEvent: the
// envelope every persisted event lives in on disk. Without it, renaming a
// field or changing the encoding passes the whole behavioral suite while
// silently invalidating existing journals (issue #23). Any intentional wire
// change must re-bless the golden and ship a CHANGELOG entry:
//
//	BBOLT_REGEN_GOLDEN=1 go test ./storage/bbolt -run TestSerializableEventWireFormat
const goldenEventPath = "testdata/golden-event.cbor"

// fixedWireEvent builds a fully-populated event with fully deterministic
// field values (fixed IDs via Parse, fixed timestamp) so the serialized
// bytes are byte-stable across runs.
func fixedWireEvent(t *testing.T) event.Event {
	t.Helper()

	eventID, err := id.ParseEventID("01J9ZQ0V5N8Y4WJ7QW2R3T4S5A")
	if err != nil {
		t.Fatalf("parse event id: %v", err)
	}

	streamID, err := id.ParseStreamID("01J9ZQ0V5N8Y4WJ7QW2R3T4S5B")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	metadata := event.NewMetadata().
		WithCustom(event.MetadataKey("tenant"), "acme").
		WithCustom(event.MetadataKey("trace"), "tr-1")

	evt, err := event.ReconstructEventWithAdoptedPayload(
		eventID,
		"user.created",
		id.StreamType("User"),
		streamID,
		3, 2,
		[]byte(`{"name":"alice","tags":["admin"]}`),
		metadata,
		time.Unix(0, 1727000000000000000).UTC(),
		codec.EncodingCBOR,
		"golden",
	)
	if err != nil {
		t.Fatalf("reconstruct event: %v", err)
	}

	return evt
}

func TestSerializableEventWireFormat(t *testing.T) {
	t.Parallel()

	data, err := serializeEvent(fixedWireEvent(t))
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	golden, readErr := os.ReadFile(goldenEventPath)
	switch {
	case os.IsNotExist(readErr):
		if os.Getenv("BBOLT_REGEN_GOLDEN") != "1" {
			t.Fatalf(
				"golden file missing: run BBOLT_REGEN_GOLDEN=1 go test ./storage/bbolt -run TestSerializableEventWireFormat",
			)
		}
		writeGolden(t, data)
	case readErr != nil:
		t.Fatalf("read golden: %v", readErr)
	case os.Getenv("BBOLT_REGEN_GOLDEN") == "1":
		writeGolden(t, data)
	default:
		if !bytes.Equal(data, golden) {
			t.Fatalf("wire format drift: serialized bytes differ from %s.\n"+
				"If this change is INTENTIONAL, re-bless with a CHANGELOG entry:\n"+
				"  BBOLT_REGEN_GOLDEN=1 go test ./storage/bbolt -run TestSerializableEventWireFormat\n"+
				"got len=%d, want len=%d", goldenEventPath, len(data), len(golden))
		}
	}
}

// TestSerializableEventEnvelopeKeys pins the exact envelope key set so a
// renamed or dropped field fails loudly even when byte comparison is
// bypassed (e.g. a codec swap that reorders output).
func TestSerializableEventEnvelopeKeys(t *testing.T) {
	t.Parallel()

	data, err := serializeEvent(fixedWireEvent(t))
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	var decoded map[string]any
	if err := unmarshalCBOR(data, &decoded); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}

	keys := make([]string, 0, len(decoded))
	for key := range decoded {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	want := []string{
		"aggregate_id",
		"aggregate_type",
		"encoding",
		"id",
		"metadata",
		"occurred_at",
		"payload",
		"schema_version",
		"type",
		"version",
	}

	if !slices.Equal(keys, want) {
		t.Fatalf("envelope key drift:\n got: %v\nwant: %v", keys, want)
	}
}

// TestSerializableEventRoundTrip proves encode(decode(encode(x))) is
// byte-identical and every envelope field survives the trip.
func TestSerializableEventRoundTrip(t *testing.T) {
	t.Parallel()

	original := fixedWireEvent(t)

	data, err := serializeEvent(original)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	decoded, err := deserializeEvent(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if decoded.ID() != original.ID() {
		t.Errorf("ID drift: got %s, want %s", decoded.ID(), original.ID())
	}

	if decoded.Type() != original.Type() {
		t.Errorf("type drift: got %s, want %s", decoded.Type(), original.Type())
	}

	if decoded.StreamID() != original.StreamID() {
		t.Errorf("stream ID drift: got %s, want %s", decoded.StreamID(), original.StreamID())
	}

	if decoded.StreamType() != original.StreamType() {
		t.Errorf("stream type drift: got %s, want %s", decoded.StreamType(), original.StreamType())
	}

	if decoded.Version().Int() != original.Version().Int() {
		t.Errorf(
			"version drift: got %d, want %d",
			decoded.Version().Int(),
			original.Version().Int(),
		)
	}

	if decoded.SchemaVersion().Int() != original.SchemaVersion().Int() {
		t.Errorf("schema version drift: got %d, want %d",
			decoded.SchemaVersion().Int(), original.SchemaVersion().Int())
	}

	if !bytes.Equal(decoded.Payload(), original.Payload()) {
		t.Errorf("payload drift: got %q, want %q", decoded.Payload(), original.Payload())
	}

	if !decoded.OccurredAt().Equal(original.OccurredAt()) {
		t.Errorf("occurred-at drift: got %s, want %s", decoded.OccurredAt(), original.OccurredAt())
	}

	if decoded.Encoding() != original.Encoding() {
		t.Errorf("encoding drift: got %s, want %s", decoded.Encoding(), original.Encoding())
	}

	if len(decoded.Metadata().Custom) != len(original.Metadata().Custom) {
		t.Errorf("metadata drift: got %d custom entries, want %d",
			len(decoded.Metadata().Custom), len(original.Metadata().Custom))
	}

	redone, err := serializeEvent(decoded)
	if err != nil {
		t.Fatalf("re-serialize: %v", err)
	}

	if !bytes.Equal(data, redone) {
		t.Error("encode(decode(encode(x))) != encode(x): serialization is not stable")
	}
}

// TestSerializableEventSchemaVersionRepresentation pins how schema_version
// rides the wire: the CBOR codec emits it on every event (json omitempty
// does not apply to CBOR encoding), as an unsigned integer.
func TestSerializableEventSchemaVersionRepresentation(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, schemaVersion int) event.Event {
		t.Helper()

		evt, err := event.ReconstructEventWithAdoptedPayload(
			id.NewEventID(),
			"user.created",
			id.StreamType("User"),
			id.NewStreamID(),
			1, schemaVersion,
			[]byte(`{}`),
			event.NewMetadata(),
			time.Unix(0, 1727000000000000000).UTC(),
			codec.EncodingCBOR,
			"golden",
		)
		if err != nil {
			t.Fatalf("reconstruct event: %v", err)
		}

		return evt
	}

	zero, err := serializeEvent(build(t, 0))
	if err != nil {
		t.Fatalf("serialize schema_version=0: %v", err)
	}

	var zeroMap map[string]any
	if err := unmarshalCBOR(zero, &zeroMap); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if _, present := zeroMap["schema_version"]; !present {
		t.Error("schema_version must always ride the wire: the CBOR codec ignores json omitempty")
	}

	nonZero, err := serializeEvent(build(t, 7))
	if err != nil {
		t.Fatalf("serialize schema_version=7: %v", err)
	}

	var nonZeroMap map[string]any
	if err := unmarshalCBOR(nonZero, &nonZeroMap); err != nil {
		t.Fatalf("decode: %v", err)
	}

	version, ok := nonZeroMap["schema_version"].(uint64)
	if !ok {
		t.Fatalf("schema_version not an unsigned integer: %T", nonZeroMap["schema_version"])
	}

	if version != 7 {
		t.Errorf("schema_version drift: got %d, want 7", version)
	}
}

func writeGolden(t *testing.T, data []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(goldenEventPath), 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}

	if err := os.WriteFile(goldenEventPath, data, 0o644); err != nil {
		t.Fatalf("write golden: %v", err)
	}

	t.Logf(
		"golden re-blessed at %s (%d bytes) — reference it in the CHANGELOG entry",
		goldenEventPath,
		len(data),
	)
}
