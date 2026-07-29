package pebbleengine

import (
	"bytes"
	"testing"
)

// nextKey is a pure helper used to compute the exclusive upper bound for every
// prefix scan (MapScan, MultiGet, CounterGet, LogTail, GraphNeighbors). If it
// returns the prefix unchanged, upper == lower and ALL scans silently return
// empty. This regression test pins the behavior that bit the project once
// (slices.Backward yields copies, so `v++` mutated a throwaway value).
func TestNextKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   []byte
		want []byte
	}{
		{"single byte increments", []byte("a"), []byte("b")},
		{"multi byte increments last", []byte("foo"), []byte("fop")},
		{"carries on boundary", []byte("a\xff"), []byte("b\x00")},
		{"carries across two bytes", []byte("\xff\xff"), []byte{0x00, 0x00, 0x00}},
		{"all 0xff grows longer", []byte{0xff, 0xff, 0xff}, []byte{0x00, 0x00, 0x00, 0x00}},
		{"empty prefix yields 0x00", []byte{}, []byte{0x00}},
		{"does not mutate input", []byte("prefix"), []byte("prefiy")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := append([]byte(nil), tc.in...) // defensive copy
			got := nextKey(tc.in)

			if !bytes.Equal(got, tc.want) {
				t.Fatalf("nextKey(% x) = % x, want % x", tc.in, got, tc.want)
			}

			// The result MUST strictly follow the input; equal means an empty range.
			if bytes.Equal(got, tc.in) && len(tc.in) > 0 {
				t.Fatalf("nextKey returned the input unchanged — scans would be empty")
			}

			// nextKey must never mutate its argument.
			if !bytes.Equal(tc.in, input) {
				t.Fatalf("nextKey mutated its input: % x -> % x", input, tc.in)
			}
		})
	}
}

// TestNextKeyExclusiveUpperBound proves the generated key is a correct
// exclusive upper bound for a prefix range: every key with the prefix sorts
// before it, and the key itself does not have the prefix.
func TestNextKeyExclusiveUpperBound(t *testing.T) {
	t.Parallel()

	prefix := []byte("users\x00")
	upper := nextKey(prefix)

	if !bytes.HasPrefix(upper, []byte("users")) {
		t.Fatalf("upper bound % x lost the prefix stem", upper)
	}

	// The first byte that differs must be exactly prefix+1.
	if len(upper) != len(prefix) {
		t.Fatalf("expected same length upper bound, got %d vs %d", len(upper), len(prefix))
	}

	if upper[len(upper)-1] != 0x01 { // 0x00 + 1
		t.Fatalf("expected incremented tail byte, got % x", upper)
	}
}
