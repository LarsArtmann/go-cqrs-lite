package codec

import (
	"encoding/json"
	"testing"
)

func TestJSONCodec_Encoding(t *testing.T) {
	t.Parallel()
	c := JSONCodec{}
	if got := c.Encoding(); got != EncodingJSON {
		t.Errorf("Encoding() = %q, want %q", got, EncodingJSON)
	}
}

func TestJSONCodec_RoundTrip(t *testing.T) {
	t.Parallel()
	c := JSONCodec{}

	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	original := payload{Name: "Alice", Age: 30}

	data, err := c.Encode(original)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	var decoded payload
	err = c.Decode(data, &decoded)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestJSONCodec_Encode_Map(t *testing.T) {
	t.Parallel()
	c := JSONCodec{}

	m := map[string]any{"key": "value", "num": float64(42)}

	data, err := c.Encode(m)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	var result map[string]any
	err = c.Decode(data, &result)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("got key=%v, want value", result["key"])
	}
}

func TestJSONCodec_Decode_InvalidJSON(t *testing.T) {
	t.Parallel()
	c := JSONCodec{}

	var v any
	err := c.Decode([]byte("not json"), &v)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRawCodec_Encoding(t *testing.T) {
	t.Parallel()
	c := RawCodec{}
	if got := c.Encoding(); got != EncodingRaw {
		t.Errorf("Encoding() = %q, want %q", got, EncodingRaw)
	}
}

func TestRawCodec_RoundTrip(t *testing.T) {
	t.Parallel()
	c := RawCodec{}

	original := []byte("hello raw bytes")

	data, err := c.Encode(original)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	var decoded []byte
	err = c.Decode(data, &decoded)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	if string(decoded) != string(original) {
		t.Errorf("round-trip mismatch: got %q, want %q", decoded, original)
	}
}

func TestRawCodec_Encode_WrongType(t *testing.T) {
	t.Parallel()
	c := RawCodec{}

	_, err := c.Encode("not bytes")
	if err == nil {
		t.Fatal("expected error for non-[]byte input")
	}
}

func TestRawCodec_Decode_WrongTarget(t *testing.T) {
	t.Parallel()
	c := RawCodec{}

	err := c.Decode([]byte("data"), "not a pointer to []byte")
	if err == nil {
		t.Fatal("expected error for non-*[]byte target")
	}
}

func TestRawCodec_Decode_IsCopy(t *testing.T) {
	t.Parallel()
	c := RawCodec{}

	original := []byte{1, 2, 3}
	data, _ := c.Encode(original)

	var decoded []byte
	_ = c.Decode(data, &decoded)

	decoded[0] = 99
	if data[0] == 99 {
		t.Error("Decode should return an independent copy")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	t.Parallel()
	codecs := map[string]Codec{
		"JSON": JSONCodec{},
		"Raw":  RawCodec{},
	}

	for name, c := range codecs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			enc := c.Encoding()
			if enc == "" {
				t.Error("Encoding() should not be empty")
			}
		})
	}
}

func TestJSONCodec_Encode_RawMessage(t *testing.T) {
	t.Parallel()
	c := JSONCodec{}

	raw := json.RawMessage(`{"already":"json"}`)
	data, err := c.Encode(raw)
	if err != nil {
		t.Fatalf("Encode(RawMessage) error: %v", err)
	}

	if string(data) != `{"already":"json"}` {
		t.Errorf("got %q", string(data))
	}
}

func TestJSONCodec_Encode_Nil(t *testing.T) {
	t.Parallel()
	c := JSONCodec{}

	data, err := c.Encode(nil)
	if err != nil {
		t.Fatalf("Encode(nil) error: %v", err)
	}

	if string(data) != "null" {
		t.Errorf("Encode(nil) = %q, want %q", string(data), "null")
	}
}

func TestJSONCodec_Decode_EmptyData(t *testing.T) {
	t.Parallel()
	c := JSONCodec{}

	var v map[string]any
	err := c.Decode([]byte{}, &v)
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}
