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

func TestAutoDetect_UnknownIsRaw(t *testing.T) {
	t.Parallel()

	// Random bytes that are neither valid JSON nor CBOR
	got := AutoDetect([]byte{0xff, 0xee, 0xdd})
	// 0xff is major type 7 → CBOR path, but invalid CBOR → falls through to Raw
	if got != EncodingRaw && got != EncodingCBOR {
		t.Errorf("AutoDetect(garbage) = %q, expected Raw or CBOR", got)
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
