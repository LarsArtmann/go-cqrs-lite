package codec

import (
	"encoding/json/v2"
	"math"
	"math/big"
	"strings"
	"testing"
)

func TestTranscodeToJSON_CBOR_Map(t *testing.T) {
	t.Parallel()

	in := map[string]any{"name": "alice", "count": 42.0}
	cborData, err := (CBORCodec{}).Encode(in)
	if err != nil {
		t.Fatalf("encode CBOR: %v", err)
	}

	out, err := TranscodeToJSON(cborData, EncodingCBOR)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode JSON: %v\njson: %s", err, out)
	}

	if got["name"] != "alice" {
		t.Errorf("name = %v, want alice", got["name"])
	}

	if got["count"] != float64(42) {
		t.Errorf("count = %v, want 42", got["count"])
	}
}

func TestTranscodeToJSON_CBOR_ToArrayStruct_StaysArray(t *testing.T) {
	t.Parallel()

	// toarray-encoded structs decode to CBOR arrays, not maps. TranscodeToJSON
	// is schema-free, so the array is preserved on the JSON side. This documents
	// the boundary: generic transcoding cannot reconstruct field names.
	type user struct {
		_     struct{} `cbor:",toarray"`
		Name  string
		Email string
	}

	cborData, err := (CBORCodec{}).Encode(user{Name: "alice", Email: "a@b.com"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	out, err := TranscodeToJSON(cborData, EncodingCBOR)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}

	var got []any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode JSON: %v\njson: %s", err, out)
	}

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2; json: %s", len(got), out)
	}

	if got[0] != "alice" || got[1] != "a@b.com" {
		t.Errorf("got = %v, want [alice a@b.com]", got)
	}
}

func TestTranscodeToJSON_JSON_Passthrough(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"already":"json"}`)

	out, err := TranscodeToJSON(payload, EncodingJSON)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}

	if string(out) != string(payload) {
		t.Errorf("JSON should pass through unchanged; got %q, want %q", out, payload)
	}
}

func TestTranscodeToJSON_Raw_Passthrough(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"maybe":"json-or-not"}`)

	out, err := TranscodeToJSON(payload, EncodingRaw)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}

	if string(out) != string(payload) {
		t.Errorf("Raw should pass through unchanged; got %q, want %q", out, payload)
	}
}

func TestTranscodeToJSON_InvalidCBOR_Error(t *testing.T) {
	t.Parallel()

	// 0xa1 = CBOR map of 1 pair, but the trailing key/value are garbage.
	badCBOR := []byte{0xa1, 0xff, 0xff}

	_, err := TranscodeToJSON(badCBOR, EncodingCBOR)
	if err == nil {
		t.Fatal("expected error for invalid CBOR, got nil")
	}

	if !strings.Contains(err.Error(), "transcode") {
		t.Errorf("error should mention transcode context; got %q", err)
	}
}

func TestTranscodeToJSON_NestedAndScalars(t *testing.T) {
	t.Parallel()

	in := map[string]any{
		"nested": map[string]any{"deep": []any{1, "two", true}},
		"flag":   true,
		"none":   nil,
	}

	cborData, err := (CBORCodec{}).Encode(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	out, err := TranscodeToJSON(cborData, EncodingCBOR)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode JSON: %v\njson: %s", err, out)
	}

	nested, ok := got["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested not a map: %T", got["nested"])
	}

	deep, ok := nested["deep"].([]any)
	if !ok || len(deep) != 3 {
		t.Fatalf("deep = %v", nested["deep"])
	}

	if got["flag"] != true {
		t.Errorf("flag = %v, want true", got["flag"])
	}

	if got["none"] != nil {
		t.Errorf("none = %v, want nil", got["none"])
	}
}

func TestTranscodeToJSON_CBORCompactCodec(t *testing.T) {
	t.Parallel()

	// CBORCompactCodec reports EncodingCBOR too (ADR-CODEC). Its wire format
	// must be decodable by the canonical decoder TranscodeToJSON uses, so the
	// two CBOR variants share one transcode path.
	in := map[string]any{"name": "compact", "count": 7.0}
	cborData, err := (CBORCompactCodec{}).Encode(in)
	if err != nil {
		t.Fatalf("encode compact CBOR: %v", err)
	}

	out, err := TranscodeToJSON(cborData, EncodingCBOR)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode JSON: %v\njson: %s", err, out)
	}

	if got["name"] != "compact" {
		t.Errorf("name = %v, want compact", got["name"])
	}

	if got["count"] != float64(7) {
		t.Errorf("count = %v, want 7", got["count"])
	}
}

// TestTranscodeToJSON_LargeNumbers exercises values at and beyond the int64
// and uint64 range. CBOR encodes uint64 values exceeding int64 max as a
// bignum (tag 2); values beyond uint64 require *big.Int. The schema-free
// generic decode path must still yield valid JSON without panicking. This
// documents the boundary for the generic decode (#20).
func TestTranscodeToJSON_LargeNumbers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		val  any
	}{
		{"int64_max", uint64(1<<63 - 1)},
		{"uint64_max", uint64(^uint64(0))},
		{"bignum_over_uint64", new(big.Int).Lsh(big.NewInt(1), 70)}, // 2^70, a real CBOR bignum
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cborData, err := (CBORCodec{}).Encode(map[string]any{"n": tc.val})
			if err != nil {
				t.Fatalf("encode: %v", err)
			}

			out, err := TranscodeToJSON(cborData, EncodingCBOR)
			if err != nil {
				t.Fatalf("transcode: %v", err)
			}

			// The output must be valid JSON. Numeric representation varies by
			// generic-decode path (float64 vs decimal number); we only assert
			// validity + key presence, since float64 loses precision above 2^53.
			var got map[string]any
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
			}

			if _, ok := got["n"]; !ok {
				t.Errorf("key n missing from %s", out)
			}
		})
	}
}

// TestTranscodeToJSON_EmptyContainers verifies that empty CBOR maps and arrays
// transcode to their JSON equivalents ("{}" and "[]").
func TestTranscodeToJSON_EmptyContainers(t *testing.T) {
	t.Parallel()

	t.Run("empty_map", func(t *testing.T) {
		t.Parallel()

		cborData, err := (CBORCodec{}).Encode(map[string]any{})
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		out, err := TranscodeToJSON(cborData, EncodingCBOR)
		if err != nil {
			t.Fatalf("transcode: %v", err)
		}

		if string(out) != "{}" {
			t.Errorf("empty map = %q, want {}", out)
		}
	})

	t.Run("empty_array", func(t *testing.T) {
		t.Parallel()

		cborData, err := (CBORCodec{}).Encode([]any{})
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		out, err := TranscodeToJSON(cborData, EncodingCBOR)
		if err != nil {
			t.Fatalf("transcode: %v", err)
		}

		if string(out) != "[]" {
			t.Errorf("empty array = %q, want []", out)
		}
	})
}

// TestTranscodeToJSON_MapKeysRoundTrip documents the key-ordering reality of
// the generic transcode path (#23): CBOR canonical encoding sorts map keys on
// the wire, but the generic decode produces a Go map[string]any whose keys are
// then re-encoded by json.Marshal. Under encoding/json/v2 the output key order
// is NOT guaranteed to be sorted or stable across runs (Go map iteration is
// randomized). So this test asserts key presence + values, not byte order.
// Callers needing deterministic key order must use event.DecodePayloadAuto[T]
// with a concrete struct type, or sort the JSON keys themselves.
func TestTranscodeToJSON_MapKeysRoundTrip(t *testing.T) {
	t.Parallel()

	in := map[string]any{"zebra": 1, "apple": 2, "mango": 3}
	cborData, err := (CBORCodec{}).Encode(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	out, err := TranscodeToJSON(cborData, EncodingCBOR)
	if err != nil {
		t.Fatalf("transcode: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}

	want := map[string]any{"zebra": float64(1), "apple": float64(2), "mango": float64(3)}
	for k, wantV := range want {
		gotV, ok := got[k]
		if !ok {
			t.Errorf("key %q missing from %s", k, out)
			continue
		}
		if gotV != wantV {
			t.Errorf("key %q = %v, want %v", k, gotV, wantV)
		}
	}
}
