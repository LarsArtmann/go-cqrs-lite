package codec

import (
	"encoding/json/v2"
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
