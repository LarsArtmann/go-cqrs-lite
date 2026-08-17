package metaengine

import (
	"encoding/json"
	"math"
	"slices"
	"testing"
)

// TestEncodeVectorBinary_Roundtrip pins bit-exact decode of the binary
// payload format — including NaN, infinities, and negative zero, which the
// legacy JSON format cannot represent at all.
func TestEncodeVectorBinary_Roundtrip(t *testing.T) {
	t.Parallel()

	cases := [][]float32{
		{},
		nil,
		{1},
		{1, -2.5, 3.75, 0, float32(math.Copysign(0, -1))},
		{math.MaxFloat32, math.SmallestNonzeroFloat32},
		{float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.NaN())},
	}

	for _, values := range cases {
		decoded, err := DecodeVectorBinary(EncodeVectorBinary(values))
		if err != nil {
			t.Fatalf("DecodeVectorBinary(%v): %v", values, err)
		}

		if len(decoded) != len(values) {
			t.Fatalf("roundtrip length = %d, want %d", len(decoded), len(values))
		}

		for i := range values {
			if math.Float32bits(decoded[i]) != math.Float32bits(values[i]) {
				t.Fatalf(
					"roundtrip[%d] = %v (%#x), want %v (%#x)",
					i,
					decoded[i],
					math.Float32bits(decoded[i]),
					values[i],
					math.Float32bits(values[i]),
				)
			}
		}
	}
}

// TestEncodeVectorBinary_Layout pins the exact wire layout: marker 'b',
// uint32 LE dimension, then little-endian float32s.
func TestEncodeVectorBinary_Layout(t *testing.T) {
	t.Parallel()

	data := EncodeVectorBinary([]float32{1, 2})
	if len(data) != 13 {
		t.Fatalf("payload length = %d, want 13", len(data))
	}

	if data[0] != 'b' {
		t.Errorf("marker = %q, want 'b'", data[0])
	}

	if dim := uint32(
		data[1],
	) | uint32(
		data[2],
	)<<8 | uint32(
		data[3],
	)<<16 | uint32(
		data[4],
	)<<24; dim != 2 {
		t.Errorf("dim = %d, want 2", dim)
	}

	if got := math.Float32frombits(
		uint32(data[5]) | uint32(data[6])<<8 | uint32(data[7])<<16 | uint32(data[8])<<24,
	); got != 1 {
		t.Errorf("float[0] = %v, want 1", got)
	}

	if got := math.Float32frombits(
		uint32(data[9]) | uint32(data[10])<<8 | uint32(data[11])<<16 | uint32(data[12])<<24,
	); got != 2 {
		t.Errorf("float[1] = %v, want 2", got)
	}
}

// TestEncodeVectorBinary_MarkerNeverCollidesWithJSON asserts the format
// sniff's safety condition: every payload a legacy JSON writer can produce
// starts with a byte other than the binary marker.
func TestEncodeVectorBinary_MarkerNeverCollidesWithJSON(t *testing.T) {
	t.Parallel()

	jsonPayloads := [][]byte{
		[]byte("[1,2,3]"),
		[]byte("null"),
		[]byte("[]"),
	}

	for _, payload := range jsonPayloads {
		if payload[0] == vectorBinaryMarker {
			t.Errorf("legacy JSON payload %q starts with the binary marker", payload)
		}
	}
}

func TestDecodeVectorBinary_Malformed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"marker only", []byte{'b'}},
		{"truncated header", []byte{'b', 2, 0, 0}},
		{"wrong marker", []byte{'x', 1, 0, 0, 0, 0, 0, 0x80, 0x3f}},
		{"dim exceeds payload", []byte{'b', 3, 0, 0, 0, 0, 0, 0x80, 0x3f}},
		{"trailing garbage", []byte{'b', 1, 0, 0, 0, 0, 0, 0x80, 0x3f, 0}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := DecodeVectorBinary(tc.data); err == nil {
				t.Fatalf("DecodeVectorBinary(%v) succeeded, want error", tc.data)
			}
		})
	}
}

// TestDecodeVectorAuto_LegacyJSON pins the upgrade contract: pre-binary rows
// (bare JSON arrays, including the `null` a nil Values slice produced) keep
// decoding through the sniffing read path.
func TestDecodeVectorAuto_LegacyJSON(t *testing.T) {
	t.Parallel()

	t.Run("array", func(t *testing.T) {
		t.Parallel()

		got, err := DecodeVectorAuto([]byte("[1,2.5,3]"))
		if err != nil {
			t.Fatalf("DecodeVectorAuto: %v", err)
		}

		if !slices.Equal(got, []float32{1, 2.5, 3}) {
			t.Errorf("decoded = %v, want [1 2.5 3]", got)
		}
	})

	t.Run("null from nil values", func(t *testing.T) {
		t.Parallel()

		got, err := DecodeVectorAuto([]byte("null"))
		if err != nil {
			t.Fatalf("DecodeVectorAuto(null): %v", err)
		}

		if len(got) != 0 {
			t.Errorf("decoded = %v, want empty", got)
		}
	})

	t.Run("invalid json errors", func(t *testing.T) {
		t.Parallel()

		if _, err := DecodeVectorAuto([]byte("[1,")); err == nil {
			t.Fatal("DecodeVectorAuto(invalid JSON) succeeded, want error")
		}
	})
}

// TestDecodeVectorAuto_CrossFormatEquivalence asserts both formats decode to
// identical vectors — the property brute-force engine scans rely on when a
// collection holds rows from before and after the format switch.
func TestDecodeVectorAuto_CrossFormatEquivalence(t *testing.T) {
	t.Parallel()

	values := make([]float32, 128)
	for i := range values {
		values[i] = float32(i) * 1.5
	}

	jsonPayload, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	fromJSON, err := DecodeVectorAuto(jsonPayload)
	if err != nil {
		t.Fatalf("DecodeVectorAuto(json): %v", err)
	}

	fromBinary, err := DecodeVectorAuto(EncodeVectorBinary(values))
	if err != nil {
		t.Fatalf("DecodeVectorAuto(binary): %v", err)
	}

	if !slices.Equal(fromJSON, fromBinary) {
		t.Fatal("JSON and binary payloads decode to different vectors")
	}
}
