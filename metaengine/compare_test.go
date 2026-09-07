package metaengine

import "testing"

// namedString simulates a domain enum like `type Status string` — the value
// form that exists before a row round-trips through JSON.
type namedString string

// namedInt simulates a domain enum like `type Level int`.
type namedInt int

// TestFilterValuesEqualNamedTypes pins cross-engine filter consistency:
// engines that keep typed structs in memory must match the same filter as
// engines that decode rows from JSON (where named types become primitives).
// Regression: FilterEq used reflect.DeepEqual, so WithFilter("status", Eq,
// Status("renamed")) matched on the memory engine but never on bbolt/badger.
func TestFilterValuesEqualNamedTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		actual   any
		expected any
		want     bool
	}{
		{"named string vs plain string", namedString("renamed"), "renamed", true},
		{"plain string vs named string", "renamed", namedString("renamed"), true},
		{"named string mismatch", namedString("renamed"), "failed", false},
		{"named int vs plain int", namedInt(2), 2, true},
		{"named int vs int64", namedInt(2), int64(2), true},
		{"named int mismatch", namedInt(2), 3, false},
		{"identical values", "a", "a", true},
		{"nil vs nil", nil, nil, true},
		{"nil vs string", nil, "a", false},
		{"string vs int", "1", 1, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := filterValuesEqual(tc.actual, tc.expected); got != tc.want {
				t.Errorf("filterValuesEqual(%v, %v) = %v, want %v",
					tc.actual, tc.expected, got, tc.want)
			}
		})
	}
}

// TestEvalFilterOpNamedTypes verifies the public operators delegate to
// value-based equality for Eq, Ne, and In.
func TestEvalFilterOpNamedTypes(t *testing.T) {
	t.Parallel()

	if !evalFilterOp(FilterEq, namedString("renamed"), "renamed") {
		t.Error("FilterEq: named string did not match plain string")
	}

	if evalFilterOp(FilterNe, namedString("renamed"), "renamed") {
		t.Error("FilterNe: named string should equal plain string")
	}

	if !evalFilterOp(FilterIn, namedString("renamed"), []any{"failed", "renamed"}) {
		t.Error("FilterIn: named string not found in plain string list")
	}

	if !matchFilter("renamed", FilterEq, namedString("renamed")) {
		t.Error("matchFilter Eq: plain string did not match named string")
	}
}
