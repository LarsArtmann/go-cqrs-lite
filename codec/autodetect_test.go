package codec

import (
	"testing"
)

func TestAutoDetect_Empty(t *testing.T) {
	t.Parallel()

	if got := AutoDetect(nil); got != EncodingRaw {
		t.Errorf("AutoDetect(nil) = %q, want %q", got, EncodingRaw)
	}

	if got := AutoDetect([]byte{}); got != EncodingRaw {
		t.Errorf("AutoDetect([]byte{}) = %q, want %q", got, EncodingRaw)
	}
}

func TestAutoDetect_JSON(t *testing.T) {
	t.Parallel()

	cases := [][]byte{
		[]byte(`{"name":"Alice"}`),
		[]byte(`[1,2,3]`),
		[]byte(`"hello"`),
		[]byte(`42`),
		[]byte(`true`),
		[]byte(`null`),
		[]byte(`-3.14`),
	}

	for _, data := range cases {
		got := AutoDetect(data)
		if got != EncodingJSON {
			t.Errorf("AutoDetect(%q) = %q, want %q", data, got, EncodingJSON)
		}
	}
}

func TestAutoDetect_CBOR(t *testing.T) {
	t.Parallel()

	// Encode known values with CBORCodec and verify detection.
	type user struct {
		Name  string
		Email string
	}

	cborData, err := CBORCodec{}.Encode(user{Name: "Alice", Email: "a@b.c"})
	if err != nil {
		t.Fatalf("CBOR Encode: %v", err)
	}

	if got := AutoDetect(cborData); got != EncodingCBOR {
		t.Errorf("AutoDetect(cborMap) = %q, want %q", got, EncodingCBOR)
	}

	// CBOR array
	arrData, err := CBORCodec{}.Encode([3]int{1, 2, 3})
	if err != nil {
		t.Fatalf("CBOR Encode array: %v", err)
	}

	if got := AutoDetect(arrData); got != EncodingCBOR {
		t.Errorf("AutoDetect(cborArray) = %q, want %q", got, EncodingCBOR)
	}
}

func TestAutoDetect_HighBytesAreCBOR(t *testing.T) {
	t.Parallel()

	// Bytes >= 0x80 are CBOR major types 4-7 and never start valid JSON.
	// AutoDetect returns CBOR even if the full payload is invalid CBOR —
	// it identifies the encoding family, not validity (documented behavior).
	cases := [][]byte{
		{0xff, 0xee, 0xdd}, // major type 7
		{0xa0},             // empty map
		{0x9f},             // stream array start
	}

	for _, data := range cases {
		if got := AutoDetect(data); got != EncodingCBOR {
			t.Errorf("AutoDetect(%v) = %q, want %q (first byte >= 0x80)", data, got, EncodingCBOR)
		}
	}
}

func TestAutoDetect_GenuinelyUnknownIsRaw(t *testing.T) {
	t.Parallel()

	// 0x1f is below 0x80, not a JSON structural start, not a valid JSON
	// token start, and not valid standalone CBOR → AutoDetect returns Raw.
	data := []byte{0x1f}
	if got := AutoDetect(data); got != EncodingRaw {
		t.Errorf("AutoDetect(%v) = %q, want %q", data, got, EncodingRaw)
	}
}

func TestSize(t *testing.T) {
	t.Parallel()

	type user struct {
		Name  string
		Email string
	}

	jsonSize, cborSize := Size(user{Name: "Alice", Email: "alice@example.com"})
	if jsonSize <= 0 {
		t.Errorf("jsonSize = %d, want > 0", jsonSize)
	}

	if cborSize <= 0 {
		t.Errorf("cborSize = %d, want > 0", cborSize)
	}

	// CBOR should be smaller for this simple struct
	if cborSize >= jsonSize {
		t.Errorf("cborSize %d >= jsonSize %d, expected CBOR to be smaller", cborSize, jsonSize)
	}
}

func TestSize_EncodeError(t *testing.T) {
	t.Parallel()

	// chan cannot be encoded by either codec
	jsonSize, cborSize := Size(make(chan int))
	if jsonSize != -1 {
		t.Errorf("jsonSize = %d, want -1 for unencodable value", jsonSize)
	}

	if cborSize != -1 {
		t.Errorf("cborSize = %d, want -1 for unencodable value", cborSize)
	}
}
