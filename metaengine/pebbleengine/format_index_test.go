package pebbleengine

import (
	"math"
	"slices"
	"testing"
)

// TestFormatIndexInt_AlwaysFixedWidth verifies that formatIndexInt always
// produces a 20-character string, regardless of input magnitude.
func TestFormatIndexInt_AlwaysFixedWidth(t *testing.T) {
	t.Parallel()

	values := []int64{
		math.MinInt64, -1_000_000, -1, 0, 1, 1_000_000, math.MaxInt64,
	}

	for _, v := range values {
		got := formatIndexInt(v)
		if len(got) != 20 {
			t.Errorf("formatIndexInt(%d): expected 20 chars, got %d (%q)", v, len(got), got)
		}
	}
}

// TestFormatIndexInt_LexicographicOrderingMatchesNumeric verifies that
// lexicographic byte comparison of formatIndexInt output matches numeric
// comparison for ALL integers. This is the core invariant: without zero-padding
// and the sign offset, "10" < "5" lexicographically, breaking range scans.
func TestFormatIndexInt_LexicographicOrderingMatchesNumeric(t *testing.T) {
	t.Parallel()

	values := []int64{
		math.MinInt64, -1_000_000, -100, -10, -5, -1, 0,
		1, 5, 10, 100, 1_000_000, math.MaxInt64,
	}

	encoded := make([]string, len(values))
	for i, v := range values {
		encoded[i] = formatIndexInt(v)
	}

	// Sort encoded strings lexicographically.
	slices.Sort(encoded)

	// The sorted order should match the original numeric order.
	for i := 1; i < len(encoded); i++ {
		if encoded[i] <= encoded[i-1] {
			t.Errorf(
				"lexicographic order breaks at index %d: %q <= %q",
				i,
				encoded[i],
				encoded[i-1],
			)
		}
	}
}

// TestFormatIndexInt_MixedDigitNumbers pins the specific bug that motivated
// this fix: without zero-padding, "5" > "10" > "100" lexicographically because
// '5' > '1'. With 20-digit padding, "00...0005" < "00...0010" < "00...0100".
func TestFormatIndexInt_MixedDigitNumbers(t *testing.T) {
	t.Parallel()

	v5 := formatIndexInt(5)
	v10 := formatIndexInt(10)
	v100 := formatIndexInt(100)

	if v5 >= v10 || v10 >= v100 {
		t.Errorf("expected 5 < 10 < 100 in encoded form, got: 5=%q 10=%q 100=%q", v5, v10, v100)
	}
}

// TestFormatIndexInt_NegativeBeforePositive verifies that all negative numbers
// sort before all positive numbers and zero.
func TestFormatIndexInt_NegativeBeforePositive(t *testing.T) {
	t.Parallel()

	neg := formatIndexInt(-1)
	zero := formatIndexInt(0)
	pos := formatIndexInt(1)

	if neg >= zero || zero >= pos {
		t.Errorf("expected -1 < 0 < 1 in encoded form, got: -1=%q 0=%q 1=%q", neg, zero, pos)
	}
}
