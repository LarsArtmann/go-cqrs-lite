package keycodec

import (
	"bytes"
	"testing"
)

// TestEncodeJSONMapValuesAreDeterministic pins the persisted-bytes contract:
// encoding/json/v2 dropped v1's implicit map-key sorting, so a plain Marshal
// of a map-typed value produces DIFFERENT bytes per run (Go map iteration
// order). Engine write paths persist these bytes — nondeterministic encoding
// means the same logical write lands as different rows/keys across runs,
// which breaks byte-level comparisons, split reads, and cache identity.
// The fix form is json.Deterministic(true) (verified 2026-10-25, rowed in
// TODO_LIST; pinned here 2026-11-03).
func TestEncodeJSONMapValuesAreDeterministic(t *testing.T) {
	t.Parallel()

	value := map[string]any{
		"alpha":   1,
		"bravo":   "two",
		"charlie": []any{3, true, nil},
		"delta":   map[string]int{"x": 1, "y": 2},
		"echo":    5.5,
		"foxtrot": "six",
		"golf":    7,
		"hotel":   map[string]any{"inner": "v"},
	}

	first := EncodeJSONErrBytesForTest(t, value)

	for i := range 50 {
		again := EncodeJSONErrBytesForTest(t, value)
		if !bytes.Equal(first, again) {
			t.Fatalf(
				"EncodeJSONErr map encoding is not deterministic: run 0 = %s, run %d = %s",
				first, i, again,
			)
		}
	}

	// Two separately-built equal maps must encode to identical bytes too.
	twin := map[string]any{
		"hotel":   map[string]any{"inner": "v"},
		"golf":    7,
		"foxtrot": "six",
		"echo":    5.5,
		"delta":   map[string]int{"y": 2, "x": 1},
		"charlie": []any{3, true, nil},
		"bravo":   "two",
		"alpha":   1,
	}

	if got := EncodeJSONErrBytesForTest(t, twin); !bytes.Equal(first, got) {
		t.Fatalf("equal maps encoded differently: %s vs %s", first, got)
	}

	// The lenient key/display path must be deterministic for the same reason:
	// storage KEYS built from map-typed values must be stable across runs.
	if a, b := EncodeJSON(value), EncodeJSON(value); !bytes.Equal(a, b) {
		t.Fatalf("EncodeJSON lenient path is not deterministic: %s vs %s", a, b)
	}
}

func EncodeJSONErrBytesForTest(t *testing.T, v any) []byte {
	t.Helper()

	b, err := EncodeJSONErr(v)
	if err != nil {
		t.Fatalf("EncodeJSONErr: %v", err)
	}

	return b
}
