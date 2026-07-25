package pebble

import (
	"math"
	"testing"
)

func TestSafeInt64_ClampsAtMaxInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   uint64
		want int64
	}{
		{"zero", 0, 0},
		{"small", 42, 42},
		{"MaxInt32", math.MaxInt32, math.MaxInt32},
		{"MaxInt64", math.MaxInt64, math.MaxInt64},
		{"MaxInt64+1", math.MaxInt64 + 1, math.MaxInt64},
		{"MaxUint64", math.MaxUint64, math.MaxInt64},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := safeInt64(tc.in)
			if got != tc.want {
				t.Errorf("safeInt64(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
