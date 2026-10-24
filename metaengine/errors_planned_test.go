package metaengine_test

import (
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestPlannedColumnTypeCompatible(t *testing.T) {
	cases := []struct {
		typ  string
		val  any
		want bool
	}{
		{"INTEGER", int64(3), true},
		{"INTEGER", float64(3), true},
		{"INTEGER", int(3), true},
		{"INTEGER", "3", false},
		{"REAL", 1.5, true},
		{"REAL", int64(1), true},
		{"REAL", "1.5", false},
		{"BIGINT", int64(3), true},
		{"BIGINT", "3", false},
		{"DOUBLE", 1.5, true}, // sqlTypeOf emits DOUBLE for float64 fields
		{"DOUBLE", int(1), true},
		{"DOUBLE", "1.5", false},
		{"FLOAT", float32(1.5), true},
		{"FLOAT", "1.5", false},
		{"INTEGER", uint8(3), true},
		{"INTEGER", float32(3), true},
		{"TEXT", "x", true},
		{"TEXT", float64(1), false},
		{"", float64(1), true}, // unknown type is permissive
		{"BLOB", "x", true},    // unknown type is permissive
		{"text", "x", true},    // case-insensitive
	}

	for _, tc := range cases {
		if got := metaengine.PlannedColumnTypeCompatible(tc.typ, tc.val); got != tc.want {
			t.Errorf("PlannedColumnTypeCompatible(%q, %T) = %v, want %v",
				tc.typ, tc.val, got, tc.want)
		}
	}
}

func TestPlannedColumnType(t *testing.T) {
	plan := metaengine.BuildLayoutPlan("c", []string{"status"}, []string{"priority"})

	if typ, ok := metaengine.PlannedColumnType(plan, "status"); !ok || typ != "TEXT" {
		t.Fatalf("status: %q %v", typ, ok)
	}

	if _, ok := metaengine.PlannedColumnType(plan, "missing"); ok {
		t.Fatal("missing column must not be found")
	}
}

func TestErrPlannedColumnTypeMismatchIsRejectionFamily(t *testing.T) {
	wrapped := errors.Join(metaengine.ErrPlannedColumnTypeMismatch)

	if !errors.Is(wrapped, metaengine.ErrPlannedColumnTypeMismatch) {
		t.Fatal("errors.Is must match through wrapping")
	}
}
