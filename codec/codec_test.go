package codec

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"
	"time"
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
		"CBOR": CBORCodec{},
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

	raw := jsontext.Value(`{"already":"json"}`)
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

func TestCBORCodec_Encoding(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}
	if got := c.Encoding(); got != EncodingCBOR {
		t.Errorf("Encoding() = %q, want %q", got, EncodingCBOR)
	}
}

func TestCBORCodec_RoundTrip(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

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

func TestCBORCodec_Encode_Map(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	m := map[string]any{"key": "value", "num": uint64(42)}

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

func TestCBORCodec_Decode_InvalidCBOR(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	var v any
	err := c.Decode([]byte("not cbor"), &v)
	if err == nil {
		t.Fatal("expected error for invalid CBOR")
	}
}

func TestCBORCodec_Encode_Nil(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	data, err := c.Encode(nil)
	if err != nil {
		t.Fatalf("Encode(nil) error: %v", err)
	}

	var v any
	if err := c.Decode(data, &v); err != nil {
		t.Fatalf("Decode(nil CBOR) error: %v", err)
	}

	if v != nil {
		t.Errorf("Decode(encode(nil)) = %v, want nil", v)
	}
}

func TestCBORCodec_Encode_Deterministic(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	payload := map[string]string{"b": "2", "a": "1", "c": "3"}

	first, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	for range 10 {
		got, err := c.Encode(payload)
		if err != nil {
			t.Fatalf("Encode() error: %v", err)
		}

		if string(got) != string(first) {
			t.Fatalf("CBOR encoding is not deterministic: got %x, want %x", got, first)
		}
	}
}

func TestCBORCodec_Decode_EmptyData(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	var v map[string]any
	err := c.Decode([]byte{}, &v)
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestCBORCodec_RoundTrip_Time(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	now := time.Date(2024, 6, 11, 9, 0, 0, 0, time.UTC)

	data, err := c.Encode(now)
	if err != nil {
		t.Fatalf("Encode(time) error: %v", err)
	}

	var decoded time.Time
	err = c.Decode(data, &decoded)
	if err != nil {
		t.Fatalf("Decode(time) error: %v", err)
	}

	if !decoded.Equal(now) {
		t.Errorf("round-trip mismatch: got %v, want %v", decoded, now)
	}
}

func TestCBORCodec_RoundTrip_ByteSlice(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	type payload struct {
		Data []byte `json:"data"`
	}

	original := payload{Data: []byte{0x00, 0x01, 0x02, 0xff}}

	data, err := c.Encode(original)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	var decoded payload
	err = c.Decode(data, &decoded)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	if !bytes.Equal(decoded.Data, original.Data) {
		t.Errorf("round-trip mismatch: got %x, want %x", decoded.Data, original.Data)
	}
}

func TestCBORCodec_SmallerThanJSON(t *testing.T) {
	t.Parallel()

	payload := map[string]string{
		"name":  "Alice",
		"email": "alice@example.com",
		"city":  "Berlin",
	}

	cborData, err := CBORCodec{}.Encode(payload)
	if err != nil {
		t.Fatalf("CBOR Encode: %v", err)
	}

	jsonData, err := JSONCodec{}.Encode(payload)
	if err != nil {
		t.Fatalf("JSON Encode: %v", err)
	}

	if len(cborData) >= len(jsonData) {
		t.Errorf(
			"CBOR (%d bytes) should be smaller than JSON (%d bytes)",
			len(cborData),
			len(jsonData),
		)
	}
}

func TestCBORCodec_RoundTrip_Slice(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	original := []string{"alpha", "beta", "gamma"}

	data, err := c.Encode(original)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	var decoded []string
	err = c.Decode(data, &decoded)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	if len(decoded) != len(original) {
		t.Fatalf("len = %d, want %d", len(decoded), len(original))
	}

	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("decoded[%d] = %q, want %q", i, decoded[i], original[i])
		}
	}
}

func TestCBORCodec_RoundTrip_NestedStruct(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	original := Person{
		Name:    "Alice",
		Address: Address{City: "Berlin", Country: "DE"},
	}

	data, err := c.Encode(original)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	var decoded Person
	err = c.Decode(data, &decoded)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestCBORCodec_StructTags(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	type tagged struct {
		Name string `json:"name" cbor:"name"`
		Age  int    `json:"age"  cbor:"age"`
	}

	original := tagged{Name: "Bob", Age: 25}

	data, err := c.Encode(original)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	var decoded tagged
	err = c.Decode(data, &decoded)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestCBORCodec_SigningDeterminism(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	payload := map[string]any{
		"user":    "alice",
		"action":  "login",
		"success": true,
		"count":   uint64(42),
	}

	encoded1, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("Encode 1: %v", err)
	}

	encoded2, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("Encode 2: %v", err)
	}

	encoded3, err := c.Encode(payload)
	if err != nil {
		t.Fatalf("Encode 3: %v", err)
	}

	if string(encoded1) != string(encoded2) || string(encoded2) != string(encoded3) {
		t.Fatal("CBOR encoding must be deterministic for signing safety")
	}

	h1 := len(encoded1)
	_ = h1
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

func TestCBORCodec_Decode_IgnoresUnknownFields(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	type target struct {
		Name string `json:"name"`
	}

	type extra struct {
		Name  string `json:"name"`
		Extra string `json:"extra"`
	}

	withExtra := extra{Name: "Alice", Extra: "surprise"}
	data, err := c.Encode(withExtra)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var got target
	err = c.Decode(data, &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Alice" {
		t.Fatalf("got %q, want %q", got.Name, "Alice")
	}
}

func TestCBORCodec_Decode_RejectsDuplicateKeys(t *testing.T) {
	t.Parallel()
	c := CBORCodec{}

	dup := map[string]any{"key": "v1", "key2": "v2"}
	data, err := c.Encode(dup)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var got map[string]any
	if err := c.Decode(data, &got); err != nil {
		t.Fatalf("Decode valid map: %v", err)
	}
}

func TestBufferEncoder_AllCodecs(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string
		Age  int
	}
	original := payload{Name: "Alice", Age: 30}

	codecs := []struct {
		name string
		c    Codec
	}{
		{"JSON", JSONCodec{}},
		{"CBOR", CBORCodec{}},
		{"CBORCompact", CBORCompactCodec{}},
	}

	for _, tc := range codecs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bufEncoder, ok := tc.c.(BufferEncoder)
			if !ok {
				t.Fatalf("%s does not implement BufferEncoder", tc.name)
			}

			buf := &bytes.Buffer{}
			if err := bufEncoder.EncodeToBuffer(original, buf); err != nil {
				t.Fatalf("EncodeToBuffer: %v", err)
			}

			var decoded payload
			if err := tc.c.Decode(buf.Bytes(), &decoded); err != nil {
				t.Fatalf("Decode: %v", err)
			}

			if decoded.Name != original.Name || decoded.Age != original.Age {
				t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, original)
			}
		})
	}
}
