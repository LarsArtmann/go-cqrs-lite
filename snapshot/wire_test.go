package snapshot_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func wireTestSnapshot(t *testing.T) snapshot.Snapshot {
	t.Helper()

	return snapshot.Snapshot{
		StreamID:   idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		StreamType: "User",
		Version:    event.Version(5),
		State:      []byte(`{"name":"Alice"}`),
		Encoding:   record.EncodingJSON,
		CreatedAt:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}
}

func TestWire_MarshalEmitsOnlyStreamKeys(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(wireTestSnapshot(t))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	out := string(data)
	for _, want := range []string{`"stream_id"`, `"stream_type"`, `"version"`, `"state"`} {
		if !strings.Contains(out, want) {
			t.Errorf("marshal output missing %s: %s", want, out)
		}
	}
	for _, banned := range []string{"aggregateId", "aggregateType"} {
		if strings.Contains(out, banned) {
			t.Errorf("marshal output still emits legacy key %s: %s", banned, out)
		}
	}
}

func TestWire_UnmarshalNewKeys(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(wireTestSnapshot(t))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got snapshot.Snapshot
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := wireTestSnapshot(t)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("roundtrip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestWire_UnmarshalLegacyAggregateKeys(t *testing.T) {
	t.Parallel()

	// Pre-v5 document spelling: camelCase aggregate keys (snapshot/store.go
	// tags before the v5 rename).
	const legacy = `{
		"aggregateId": "01HK1540X0841Y0A6BSX1VKR95",
		"aggregateType": "User",
		"version": 5,
		"state": "eyJuYW1lIjoiQWxpY2UifQ==",
		"encoding": 1,
		"createdAt": "2026-06-01T12:00:00Z"
	}`

	var got snapshot.Snapshot
	if err := json.Unmarshal([]byte(legacy), &got); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}

	want := wireTestSnapshot(t)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("legacy decode mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestWire_NewKeysWinOverLegacy(t *testing.T) {
	t.Parallel()

	// A document carrying both spellings must resolve to the honest one.
	const mixed = `{"stream_id":"01HK1540X0841Y0A6BSX1VKR95","stream_type":"User","aggregateId":"01HGW5FPJPYK5RE8ACZDesWMY2","aggregateType":"Order","version":1,"state":"aGk=","createdAt":"2026-06-01T12:00:00Z"}`

	var got snapshot.Snapshot
	if err := json.Unmarshal([]byte(mixed), &got); err != nil {
		t.Fatalf("unmarshal mixed: %v", err)
	}

	if got.StreamType != "User" {
		t.Errorf("expected stream_type to win, got %+v", got)
	}
}

func TestWire_CBORRoundTripUnaffectedByTagRename(t *testing.T) {
	t.Parallel()

	// Canonical CBOR keys structs by Go field name, so the wire bytes never
	// carried the aggregate vocabulary: pre-v5 and post-rename CBOR payloads
	// are interchangeable.
	want := wireTestSnapshot(t)

	encoded, err := codec.CBORCodec{}.Encode(want)
	if err != nil {
		t.Fatalf("cbor encode: %v", err)
	}

	var got snapshot.Snapshot
	if err := (codec.CBORCodec{}).Decode(encoded, &got); err != nil {
		t.Fatalf("cbor decode: %v", err)
	}

	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("cbor created-at mismatch: got %v want %v", got.CreatedAt, want.CreatedAt)
	}

	got.CreatedAt, want.CreatedAt = time.Time{}, time.Time{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cbor roundtrip mismatch:\n got %+v\nwant %+v", got, want)
	}
}
