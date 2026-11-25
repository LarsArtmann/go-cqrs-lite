package metaengine

import "testing"

type namedPriority int

type namedScore float64

// Named numeric types must order numerically, not by their printed form
// (where "10" sorts before "9").
func TestCompareValue_NamedNumericTypesOrderByValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		left, right any
		want        int
	}{
		{"named int vs unnamed int", namedPriority(10), 9, 1},
		{"unnamed int vs named int", 9, namedPriority(10), -1},
		{"named int vs named int", namedPriority(2), namedPriority(10), -1},
		{"named int vs decoded float64", namedPriority(10), float64(9), 1},
		{"named float vs decoded float64", namedScore(0.5), 0.25, 1},
		{"equal named and unnamed", namedPriority(7), int64(7), 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := compareValue(tc.left, tc.right); got != tc.want {
				t.Fatalf("compareValue(%v, %v) = %d, want %d", tc.left, tc.right, got, tc.want)
			}
		})
	}
}

func TestMatchFilter_NamedNumericRangeFilters(t *testing.T) {
	t.Parallel()

	if !matchFilter(namedPriority(10), FilterLt, 50) {
		t.Fatal("named 10 must be less than 50")
	}

	if matchFilter(namedPriority(10), FilterLt, 9) {
		t.Fatal("named 10 must not be less than 9 (printed-form comparison would say it is)")
	}

	if !matchFilter(namedPriority(2), FilterGt, 1.5) {
		t.Fatal("named 2 must be greater than 1.5")
	}
}

func TestCompareValue_UnrelatedTypesKeepPrintedFormFallback(t *testing.T) {
	t.Parallel()

	type plain struct{ A int }

	if got := compareValue(plain{A: 1}, plain{A: 2}); got != -1 {
		t.Fatalf("struct fallback = %d, want -1", got)
	}
}
