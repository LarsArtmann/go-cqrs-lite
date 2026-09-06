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

func TestWire_CBORCarriesNewKeys(t *testing.T) {
	t.Parallel()

	// fxamacker/cbor v2.9 falls back to the json tag key when no cbor key is
	// present, so the tag rename moves the CBOR map keys as well: writers
	// must emit the honest stream keys from now on.
	data, err := codec.CBORCodec{}.Encode(wireTestSnapshot(t))
	if err != nil {
		t.Fatalf("cbor encode: %v", err)
	}

	var keys map[string]any
	if err := codec.CBORDecMode().Unmarshal(data, &keys); err != nil {
		t.Fatalf("cbor decode map: %v", err)
	}

	for _, want := range []string{"stream_id", "stream_type"} {
		if _, ok := keys[want]; !ok {
			t.Errorf("CBOR map missing new key %q: %v", want, keys)
		}
	}
	for _, banned := range []string{"aggregateId", "aggregateType"} {
		if _, ok := keys[banned]; ok {
			t.Errorf("CBOR map still emits legacy key %q: %v", banned, keys)
		}
	}
}

func TestWire_CBORRoundTripAndLegacyKeys(t *testing.T) {
	t.Parallel()

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

	// Pre-v5 CBOR bytes carry the aggregateId/aggregateType keys; the
	// decode-only fallback must restore the identity.
	legacy, err := codec.CBOREncMode().Marshal(struct {
		StreamID   []byte          `json:"aggregateId"`
		StreamType string          `json:"aggregateType"`
		Version    int             `json:"version"`
		State      []byte          `json:"state"`
		Encoding   record.Encoding `json:"encoding,omitempty"`
		CreatedAt  time.Time       `json:"createdAt"`
	}{
		// StreamID encodes as a CBOR byte string: pre-v5 writers serialized
		// id.StreamID through its BinaryMarshaler, and that byte-string form
		// is what snapshotWireLegacy must decode.
		StreamID:   []byte(want.StreamID.String()),
		StreamType: string(want.StreamType),
		Version:    want.Version.Int(),
		State:      want.State,
		Encoding:   want.Encoding,
		CreatedAt:  want.CreatedAt,
	})
	if err != nil {
		t.Fatalf("cbor marshal legacy: %v", err)
	}

	var fromLegacy snapshot.Snapshot
	if err := (codec.CBORCodec{}).Decode(legacy, &fromLegacy); err != nil {
		t.Fatalf("cbor decode legacy: %v", err)
	}

	if fromLegacy.StreamID != want.StreamID || fromLegacy.StreamType != want.StreamType {
		t.Errorf("legacy CBOR identity mismatch: got %s/%s", fromLegacy.StreamType, fromLegacy.StreamID)
	}
	if fromLegacy.Version != want.Version || !fromLegacy.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("legacy CBOR payload mismatch: %+v", fromLegacy)
	}
}
