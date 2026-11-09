package compat

import "testing"

func TestClassString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		class Class
		want  string
	}{
		{ClassUnchanged, "unchanged"},
		{ClassAdditive, "additive"},
		{ClassNeedsVersionBump, "needs-version-bump"},
		{ClassIncomparable, "incomparable"},
		{Class(99), "unknown"},
	}

	for _, tc := range cases {
		if got := tc.class.String(); got != tc.want {
			t.Errorf("Class(%d).String() = %q, want %q", tc.class, got, tc.want)
		}
	}
}

func TestClassIsBreaking(t *testing.T) {
	t.Parallel()

	cases := []struct {
		class Class
		want  bool
	}{
		{ClassUnchanged, false},
		{ClassAdditive, false},
		{ClassNeedsVersionBump, true},
		{ClassIncomparable, true},
	}

	for _, tc := range cases {
		if got := tc.class.IsBreaking(); got != tc.want {
			t.Errorf("Class(%s).IsBreaking() = %v, want %v", tc.class, got, tc.want)
		}
	}
}

func TestClassWorst(t *testing.T) {
	t.Parallel()

	all := []Class{ClassUnchanged, ClassAdditive, ClassNeedsVersionBump, ClassIncomparable}

	for _, left := range all {
		for _, right := range all {
			got := left.Worst(right)
			if got < left || got < right {
				t.Fatalf("Worst(%s, %s) = %s, less severe than an input", left, right, got)
			}

			if right.Worst(left) != got {
				t.Fatalf("Worst is not commutative: %s/%s", left, right)
			}
		}
	}
}
