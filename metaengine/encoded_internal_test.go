package metaengine

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type storedEvent struct {
	ID   string
	Name string
	At   time.Time
}

type storedView struct {
	ID   string
	Name string
}

type storedKey struct {
	ID string
}

func newStoredStore(t *testing.T) *Store {
	t.Helper()

	query := Query[storedKey, storedView](
		"stored",
		OnRecordTyped(
			"stored.event",
			storedEvent{},
			func(_ record.Record, e storedEvent) (string, storedView) {
				return e.ID, storedView{ID: e.ID, Name: e.Name}
			},
		),
	)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
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

// TestApplyEncoded_StoresJSONPayloadVerbatim pins the JSON contract: the
// EventLog receives the caller's bytes unchanged (the marker is jsontext.Value,
// not a re-marshaled value), so replay and wire assertions keep comparing the
// same bytes as before the codec work.
func TestApplyEncoded_StoresJSONPayloadVerbatim(t *testing.T) {
	t.Parallel()

	store := newStoredStore(t)
	log := NewEventLog()
	WithEventLog(store, log)

	payload := []byte(`{"ID":"e1","Name":"verbatim","At":"2026-01-01T00:00:00Z"}`)

	if err := store.ApplyEncoded(context.Background(), "stored.event", payload); err != nil {
		t.Fatalf("ApplyEncoded: %v", err)
	}

	events := log.Events()
	if len(events) != 1 {
		t.Fatalf("logged %d events, want 1", len(events))
	}

	raw, ok := events[0].Payload.(jsontext.Value)
	if !ok {
		t.Fatalf("EventLog payload type = %T, want jsontext.Value", events[0].Payload)
	}

	if !bytes.Equal(raw, payload) {
		t.Fatalf("EventLog payload = %q, want the caller's bytes %q", raw, payload)
	}
}

// TestApplyEncoded_StoresCBORMarkerVerbatim pins the CBOR contract: the marker
// carries the producer's bytes and its codec, never a transcoded copy.
func TestApplyEncoded_StoresCBORMarkerVerbatim(t *testing.T) {
	t.Parallel()

	store := newStoredStore(t)
	log := NewEventLog()
	WithEventLog(store, log)

	payload, err := codec.CBORCodec{}.Encode(storedEvent{
		ID: "e2", Name: "cbor", At: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CBOR encode: %v", err)
	}

	if err := store.ApplyEncoded(context.Background(), "stored.event", payload); err != nil {
		t.Fatalf("ApplyEncoded: %v", err)
	}

	events := log.Events()
	if len(events) != 1 {
		t.Fatalf("logged %d events, want 1", len(events))
	}

	marker, ok := events[0].Payload.(encodedPayload)
	if !ok {
		t.Fatalf("EventLog payload type = %T, want encodedPayload", events[0].Payload)
	}

	if marker.encoding != codec.EncodingCBOR {
		t.Fatalf("marker encoding = %q, want cbor", marker.encoding)
	}

	if !bytes.Equal(marker.raw, payload) {
		t.Fatal("marker bytes differ from the caller's payload")
	}
}

// TestApplyEncoded_EncodingsConverge proves the ingress is a representation
// choice, not a semantic one: the same event JSON-encoded and CBOR-encoded
// produces the same projection state.
func TestApplyEncoded_EncodingsConverge(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	event := storedEvent{
		ID: "e3", Name: "converge", At: time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC),
	}

	jsonPayload, err := codec.JSONCodec{}.Encode(event)
	if err != nil {
		t.Fatalf("JSON encode: %v", err)
	}

	cborPayload, err := codec.CBORCodec{}.Encode(event)
	if err != nil {
		t.Fatalf("CBOR encode: %v", err)
	}

	jsonStore := newStoredStore(t)
	cborStore := newStoredStore(t)

	if err := jsonStore.ApplyEncoded(ctx, "stored.event", jsonPayload); err != nil {
		t.Fatalf("JSON ApplyEncoded: %v", err)
	}

	if err := cborStore.ApplyEncoded(ctx, "stored.event", cborPayload); err != nil {
		t.Fatalf("CBOR ApplyEncoded: %v", err)
	}

	jsonView, err := ExecuteTyped[storedKey, storedView](ctx, jsonStore, storedKey{ID: "e3"})
	if err != nil {
		t.Fatalf("JSON ExecuteTyped: %v", err)
	}

	cborView, err := ExecuteTyped[storedKey, storedView](ctx, cborStore, storedKey{ID: "e3"})
	if err != nil {
		t.Fatalf("CBOR ExecuteTyped: %v", err)
	}

	if jsonView != cborView {
		t.Fatalf("views differ by encoding: json=%+v cbor=%+v", jsonView, cborView)
	}
}

// TestEncodePayload_DetectsShapes pins the sniff matrix: every JSON and CBOR
// shape the library can produce must land on the right marker.
func TestEncodePayload_DetectsShapes(t *testing.T) {
	t.Parallel()

	cbor := func(t *testing.T, v any) []byte {
		t.Helper()

		data, err := codec.CBORCodec{}.Encode(v)
		if err != nil {
			t.Fatalf("CBOR encode: %v", err)
		}

		return data
	}

	cases := []struct {
		name    string
		payload []byte
		json    bool
	}{
		{name: "json object", payload: []byte(`{"a":1}`), json: true},
		{name: "json array", payload: []byte(`[1,2]`), json: true},
		{name: "json string", payload: []byte(`"text"`), json: true},
		{name: "json number", payload: []byte(`42`), json: true},
		{name: "json bool", payload: []byte(`true`), json: true},
		{name: "json null", payload: []byte(`null`), json: true},
		{name: "cbor map", payload: cbor(t, map[string]int{"a": 1})},
		{name: "cbor array", payload: cbor(t, []int{1, 2})},
		{name: "cbor text", payload: cbor(t, "text")},
		{name: "cbor uint", payload: cbor(t, uint64(42))},
		{name: "cbor negative", payload: cbor(t, int64(-7))},
		{name: "cbor float", payload: cbor(t, 1.5)},
		{name: "cbor bool", payload: cbor(t, true)},
		{name: "cbor null", payload: cbor(t, nil)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			marker, err := encodePayload(tc.payload, "")
			if err != nil {
				t.Fatalf("encodePayload: %v", err)
			}

			_, isJSON := marker.(jsontext.Value)
			if isJSON != tc.json {
				t.Fatalf("marker type = %T, want json=%v", marker, tc.json)
			}

			if tc.json {
				if !bytes.Equal(marker.(jsontext.Value), tc.payload) {
					t.Fatal("JSON marker does not carry the input bytes")
				}

				return
			}

			encoded, ok := marker.(encodedPayload)
			if !ok {
				t.Fatalf("marker type = %T, want encodedPayload", marker)
			}

			if encoded.encoding != codec.EncodingCBOR {
				t.Fatalf("marker encoding = %q, want cbor", encoded.encoding)
			}
		})
	}
}

func TestEncodePayload_RejectsEmptyAndUnknown(t *testing.T) {
	t.Parallel()

	if _, err := encodePayload(nil, ""); err == nil {
		t.Fatal("nil payload: expected an error")
	}

	if _, err := encodePayload([]byte{}, ""); err == nil {
		t.Fatal("empty payload: expected an error")
	}

	// 0x1c is a reserved CBOR additional-information value: neither JSON nor
	// decodable CBOR, so detection must refuse instead of guessing.
	if _, err := encodePayload([]byte{0x1c, 0x1c}, ""); err == nil {
		t.Fatal("unrecognized payload: expected an error")
	}
}

func TestCodecEncodingOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   record.Encoding
		want codec.Encoding
	}{
		{in: record.EncodingUnknown, want: ""},
		{in: record.EncodingJSON, want: codec.EncodingJSON},
		{in: record.EncodingCBOR, want: codec.EncodingCBOR},
		{in: record.Encoding(99), want: ""},
	}

	for _, tc := range cases {
		if got := codecEncodingOf(tc.in); got != tc.want {
			t.Errorf("codecEncodingOf(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// FuzzEncodePayload asserts the ingress never panics and never returns an
// unexpected marker type for arbitrary bytes.
func FuzzEncodePayload(f *testing.F) {
	f.Add([]byte(`{"a":1}`))
	f.Add([]byte(`"text"`))
	f.Add([]byte{0xa1, 0x61, 0x61, 0x01})
	f.Add([]byte{})
	f.Add([]byte{0x1c})

	f.Fuzz(func(t *testing.T, payload []byte) {
		marker, err := encodePayload(payload, "")
		if err != nil {
			if marker != nil {
				t.Fatalf("error returned with a non-nil marker %T", marker)
			}

			return
		}

		switch marker.(type) {
		case jsontext.Value, encodedPayload:
		default:
			t.Fatalf("unexpected marker type %T", marker)
		}
	})
}
