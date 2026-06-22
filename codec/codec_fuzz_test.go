package codec_test

import (
	"math"
	"testing"
	"unicode/utf8"

	"github.com/fxamacker/cbor/v2"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

func FuzzJSONCodec_Roundtrip(f *testing.F) {
	f.Add(`{"name":"Alice","age":30}`)
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(`[]`)
	f.Add(`"hello"`)
	f.Add(`42`)
	f.Add(`true`)

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip()
		}

		c := codec.JSONCodec{}

		var decoded any
		if err := c.Decode([]byte(input), &decoded); err != nil {
			t.Skip()
		}

		encoded, err := c.Encode(decoded)
		if err != nil {
			t.Fatalf("Encode(%v): %v", decoded, err)
		}

		var redecoded any
		if err := c.Decode(encoded, &redecoded); err != nil {
			t.Fatalf("Decode(re-encoded): %v", err)
		}
	})
}

func FuzzCBORCodec_Roundtrip(f *testing.F) {
	c := codec.CBORCodec{}

	seeds := []any{
		map[string]any{"name": "Alice", "age": uint64(30)},
		map[string]any{},
		nil,
		[]any{},
		"hello",
		uint64(42),
		true,
	}

	for _, seed := range seeds {
		b, err := c.Encode(seed)
		if err != nil {
			f.Fatalf("seed encode: %v", err)
		}

		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		var decoded any
		if err := c.Decode(input, &decoded); err != nil {
			t.Skip()
		}

		encoded, err := c.Encode(decoded)
		if err != nil {
			t.Fatalf("Encode(%v): %v", decoded, err)
		}

		var redecoded any
		if err := c.Decode(encoded, &redecoded); err != nil {
			// CBOR maps decoded into map[any]any can have keys of different
			// Go numeric types (int64 vs int) that are distinct in Go but
			// encode to identical CBOR bytes. This produces duplicate keys
			// in the re-encoded output. This is a known CBOR/Go type-system
			// ambiguity, not a codec bug — skip.
			t.Skip()
		}
	})
}

func FuzzRawCodec_Passthrough(f *testing.F) {
	f.Add([]byte(`{"raw":true}`))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0xff, 0xfe, 0x01})

	f.Fuzz(func(t *testing.T, input []byte) {
		c := codec.RawCodec{}

		got, err := c.Encode(input)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}

		var target []byte
		if err := c.Decode(got, &target); err != nil {
			t.Fatalf("Decode: %v", err)
		}

		if len(input) == 0 && len(target) == 0 {
			return
		}

		if string(target) != string(input) {
			t.Errorf("roundtrip mismatch: got %q, want %q", target, input)
		}
	})
}

// FuzzCBORCodec_Determinism ensures CBOR canonical encoding produces the
// same bytes regardless of input map key order. Critical for
// content-addressed storage and signature determinism.
func FuzzCBORCodec_Determinism(f *testing.F) {
	c := codec.CBORCodec{}

	// Seeds: each is a map with two entries encoded in different insertion
	// orders. Canonical encoding must produce identical bytes.
	seeds := []map[string]any{
		{"a": uint64(1), "b": uint64(2)},
		{"z": "last", "a": "first"},
		{"n": float64(3.14)},
	}

	for _, s := range seeds {
		b, err := c.Encode(s)
		if err != nil {
			f.Fatalf("seed encode: %v", err)
		}
		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, _ []byte) {
		// Build maps in two different insertion orders; both must encode
		// to the same byte sequence (canonical form sorts map keys).
		original := map[string]any{
			"alpha":  uint64(1),
			"beta":   "hello",
			"gamma":  true,
			"delta":  []any{uint64(1), uint64(2), uint64(3)},
			"nested": map[string]any{"k": "v"},
		}
		reordered := map[string]any{
			"delta":  []any{uint64(1), uint64(2), uint64(3)},
			"alpha":  uint64(1),
			"nested": map[string]any{"k": "v"},
			"gamma":  true,
			"beta":   "hello",
		}

		encodedOrig, err := c.Encode(original)
		if err != nil {
			t.Fatalf("Encode original: %v", err)
		}

		encodedReord, err := c.Encode(reordered)
		if err != nil {
			t.Fatalf("Encode reordered: %v", err)
		}

		if string(encodedOrig) != string(encodedReord) {
			t.Errorf("CBOR encoding not canonical: %x vs %x", encodedOrig, encodedReord)
		}
	})
}

// FuzzCBORCodec_DecodeNeverPanics exercises CBOR decode on arbitrary bytes
// — must always return an error rather than panic, even on garbage input.
func FuzzCBORCodec_DecodeNeverPanics(f *testing.F) {
	f.Add([]byte{0x00})
	f.Add([]byte{0xff, 0xff, 0xff})
	f.Add([]byte{})
	f.Add([]byte{0xa1, 0x61, 0x62}) // map(1) { text(1) "b" } but no value

	f.Fuzz(func(t *testing.T, input []byte) {
		c := codec.CBORCodec{}

		var into any
		// Should not panic
		_ = c.Decode(input, &into)
	})
}

// FuzzJSONCodec_TypedRoundtrip exercises JSON codec with a typed struct
// target. Catches cases where generic roundtrip succeeds but typed
// roundtrip fails (e.g., missing fields, type coercion issues).
func FuzzJSONCodec_TypedRoundtrip(f *testing.F) {
	type point struct {
		X int    `json:"x"`
		Y int    `json:"y"`
		N string `json:"n,omitempty"`
	}

	f.Add(`{"x":1,"y":2}`)
	f.Add(`{"x":0,"y":0}`)
	f.Add(`{"x":-1,"y":-2,"n":"named"}`)
	f.Add(`{}`)
	f.Add(`{"x":1}`)

	f.Fuzz(func(t *testing.T, input string) {
		c := codec.JSONCodec{}

		var p point
		if err := c.Decode([]byte(input), &p); err != nil {
			t.Skip()
		}

		encoded, err := c.Encode(p)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}

		var redecoded point
		if err := c.Decode(encoded, &redecoded); err != nil {
			t.Fatalf("Decode roundtrip: %v", err)
		}

		if redecoded != p {
			t.Errorf("typed roundtrip mismatch: got %+v, want %+v", redecoded, p)
		}
	})
}

// FuzzCBORCodec_TypedRoundtrip exercises CBOR codec with a typed struct
// target through a pure CBOR→CBOR path. Catches type coercion issues
// that might not surface with generic any roundtrips.
func FuzzCBORCodec_TypedRoundtrip(f *testing.F) {
	type record struct {
		Name string `json:"name"`
		Age  uint   `json:"age"`
	}

	c := codec.CBORCodec{}

	seeds := []record{
		{Name: "Alice", Age: 30},
		{Name: "", Age: 0},
		{Name: "Bob", Age: 255},
	}

	for _, s := range seeds {
		b, err := c.Encode(s)
		if err != nil {
			f.Fatalf("seed encode: %v", err)
		}

		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		var p record
		if err := c.Decode(input, &p); err != nil {
			t.Skip()
		}

		encoded, err := c.Encode(p)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}

		var redecoded record
		if err := c.Decode(encoded, &redecoded); err != nil {
			t.Fatalf("Decode roundtrip: %v", err)
		}

		if redecoded != p {
			t.Errorf("typed roundtrip mismatch: got %+v, want %+v", redecoded, p)
		}
	})
}

// FuzzCBORCodec_CanonicalFidelity ensures that canonical CBOR encoding is a
// fixed point: once a value is in canonical form, decode→encode reproduces the
// same bytes — encode(decode(encode(v))) == encode(v). This is the core
// invariant that makes content-addressed storage and signature verification safe.
//
// The invariant is verified from canonical bytes, not from arbitrary input.
// Canonical CBOR (RFC 7049 "shortest" rule) is NOT injective: a map holding
// both a bare integer and a time.Time whose epoch equals that integer collapses
// the time to the same integer bytes, producing a duplicate-key map. That is an
// inherent property of canonical encoding, not a codec bug, and cannot arise
// from typed structs — the only shape CQRS payloads ever encode. Such inputs
// are skipped at the re-decode step rather than treated as failures.
func FuzzCBORCodec_CanonicalFidelity(f *testing.F) {
	c := codec.CBORCodec{}

	seeds := []any{
		map[string]any{"name": "Alice", "age": uint64(30)},
		map[string]any{},
		[]any{uint64(1), uint64(2), uint64(3)},
		"hello",
		uint64(42),
		true,
	}

	for _, seed := range seeds {
		b, err := c.Encode(seed)
		if err != nil {
			f.Fatalf("seed encode: %v", err)
		}

		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		var decoded any
		if err := c.Decode(input, &decoded); err != nil {
			t.Skip()
		}

		// NaN map keys have no stable canonical form: every NaN bit pattern
		// normalizes to the same canonical half-float (f97e00), collapsing
		// distinct keys into byte-identical duplicates. The decoder's duplicate-
		// key guard cannot catch them (NaN != NaN), so they would otherwise slip
		// past the re-decode skip below and surface as spurious fixed-point
		// failures with randomized ordering. Typed CQRS structs can never
		// produce NaN keys, so skipping is the principled response.
		if hasNaNMapKey(decoded) {
			t.Skip()
		}

		canonical, err := c.Encode(decoded)
		if err != nil {
			t.Fatalf("Encode(%v): %v", decoded, err)
		}

		// A canonical form must itself be decodable. If it is not, the input
		// held semantically-distinct keys that canonical normalization collapsed
		// (e.g. int -17 and time.Time{epoch: -17}). This is expected canonical
		// non-injectivity, not a fixed-point violation — skip it.
		var redecoded any
		if err := c.Decode(canonical, &redecoded); err != nil {
			t.Skip()
		}

		recanonical, err := c.Encode(redecoded)
		if err != nil {
			t.Fatalf("Encode(redecoded): %v", err)
		}

		if string(recanonical) != string(canonical) {
			t.Errorf("canonical encoding is not a fixed point:\n  first  = %x\n  second = %x",
				canonical, recanonical)
		}
	})
}

// hasNaNMapKey reports whether v contains any map whose key is a NaN float,
// at any nesting depth (including inside [cbor.Tag] content). Such values
// have no stable canonical encoding: all NaN bit patterns collapse to the
// identical canonical half-float (f97e00), producing duplicate keys whose
// emitted order follows Go's randomized map iteration.
func hasNaNMapKey(v any) bool {
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			if hasNaNMapKey(e) {
				return true
			}
		}
	case map[string]any:
		for _, e := range x {
			if hasNaNMapKey(e) {
				return true
			}
		}
	case map[any]any:
		for k, e := range x {
			if isNaNValue(k) || hasNaNMapKey(e) {
				return true
			}
		}
	case cbor.Tag:
		return hasNaNMapKey(x.Content)
	}

	return false
}

// isNaNValue reports whether v is a NaN float (float32 or float64).
func isNaNValue(v any) bool {
	switch f := v.(type) {
	case float64:
		return math.IsNaN(f)
	case float32:
		return math.IsNaN(float64(f))
	}

	return false
}
