package record_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

// TestPayload_TriStateRoundTriips pins review R4's state table: presence is
// three-plus honest values, not one conflated zero. Each state survives a
// Record JSON round trip (the ADR-0111 untagged shape).
func TestPayload_TriStateRoundTrips(t *testing.T) {
	t.Parallel()

	states := []struct {
		name string
		in   record.Record
	}{
		{
			name: "absent (zero Payload)",
			in:   record.Record{Type: "user.created"},
		},
		{
			name: "opaque (data, no stamp at this layer)",
			in: record.Record{
				Type:    "user.created",
				Payload: record.PayloadFrom([]byte{0x01, 0x02}, record.EncodingUnknown),
			},
		},
		{
			name: "wrapped (ADR-0044 envelope stamp)",
			in: record.Record{
				Type:    "user.created",
				Payload: record.PayloadFrom([]byte(`{"e":"env"}`), record.EncodingEnvelope),
			},
		},
		{
			name: "stamped json",
			in: record.Record{
				Type:    "user.created",
				Payload: record.PayloadFrom([]byte(`{"n":1}`), record.EncodingJSON),
			},
		},
		{
			name: "stamped cbor",
			in: record.Record{
				Type:    "user.created",
				Payload: record.PayloadFrom([]byte{0xa1}, record.EncodingCBOR),
			},
		},
	}

	for _, st := range states {
		t.Run(st.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(st.in)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var decoded record.Record
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			if decoded.Payload.Enc != st.in.Payload.Enc {
				t.Errorf("Enc = %v, want %v", decoded.Payload.Enc, st.in.Payload.Enc)
			}

			if string(decoded.Payload.Data) != string(st.in.Payload.Data) {
				t.Errorf("Data = %x, want %x", decoded.Payload.Data, st.in.Payload.Data)
			}

			if decoded.Payload.IsZero() != st.in.Payload.IsZero() {
				t.Errorf("IsZero = %v, want %v", decoded.Payload.IsZero(), st.in.Payload.IsZero())
			}
		})
	}
}

func TestPayloadFrom(t *testing.T) {
	t.Parallel()

	if got := record.PayloadFrom(nil, record.EncodingJSON); !got.IsZero() {
		t.Errorf("PayloadFrom(nil, json) = %v, want the absent state", got)
	}

	if got := record.PayloadFrom([]byte{}, record.EncodingCBOR); !got.IsZero() {
		t.Errorf("PayloadFrom(empty, cbor) = %v, want the absent state", got)
	}

	got := record.PayloadFrom([]byte("x"), record.EncodingUnknown)
	if got.IsZero() {
		t.Error("PayloadFrom(data, unknown) must not be absent (it is opaque)")
	}

	if got.Enc != record.EncodingUnknown || string(got.Data) != "x" {
		t.Errorf("PayloadFrom(data, unknown) = %v, want opaque {x unknown}", got)
	}
}

func TestPayload_EncodingEnvelopeStamp(t *testing.T) {
	t.Parallel()

	if got := record.EncodingEnvelope.String(); got != "envelope" {
		t.Errorf("EncodingEnvelope.String() = %q, want envelope", got)
	}

	if _, err := record.ParseEncoding("envelope"); err == nil {
		t.Error("ParseEncoding must not accept envelope — it is a wrapping fact, not a codec name")
	}
}
