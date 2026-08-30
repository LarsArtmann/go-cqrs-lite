package quic

import (
	"math"
	"reflect"
	"testing"
)

// TestNormalizeAny pins the CBOR decode-type restoration for op keys/values.
// CBOR major type 0 (non-negative int) decodes into uint64 when the target is
// any, and major type 1 (negative) into int64 — without normalization a Go
// int(5) round-trips as uint64(5) and consumers comparing with == or decoding
// into typed fields see the wrong type.
func TestNormalizeAny(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{"small uint64", uint64(5), int(5)},
		{"zero", uint64(0), int(0)},
		{"max int64 stays int", uint64(math.MaxInt64), int(math.MaxInt64)},
		{"beyond max int64 stays uint64", uint64(math.MaxInt64) + 1, uint64(math.MaxInt64) + 1},
		{"negative int64", int64(-7), int(-7)},
		{"min int64", int64(math.MinInt64), int(math.MinInt64)},
		{"plain int untouched", int(9), int(9)},
		{"string untouched", "k", "k"},
		{"bool untouched", true, true},
		{"float untouched", 3.5, 3.5},
		{"nil untouched", nil, nil},
		{
			"nested slice",
			[]any{uint64(1), int64(-2), []any{uint64(3)}},
			[]any{int(1), int(-2), []any{int(3)}},
		},
		{
			"nested map",
			map[string]any{"a": uint64(1), "b": map[string]any{"c": int64(-4)}},
			map[string]any{"a": int(1), "b": map[string]any{"c": int(-4)}},
		},
		{
			"slice of maps",
			[]any{map[string]any{"k": uint64(6)}},
			[]any{map[string]any{"k": int(6)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAny(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("normalizeAny(%#v) = %#v (%T), want %#v (%T)",
					tt.in, got, got, tt.want, tt.want)
			}
		})
	}
}

// TestNormalizeAny_IntRoundTrip proves the consumer-facing property: a value
// that went in as Go int compares equal (type AND value) to what comes back.
func TestNormalizeAny_IntRoundTrip(t *testing.T) {
	var in any = int(42)

	// Simulate the CBOR decode the transport performs: any-typed target.
	var decoded any = uint64(42)

	if normalizeAny(decoded) != in {
		t.Errorf("round trip: %#v != %#v", normalizeAny(decoded), in)
	}
}
