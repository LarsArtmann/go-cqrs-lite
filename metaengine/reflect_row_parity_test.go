package metaengine_test

import (
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Issue #63: the closure-fallback row resolution matched Go field names
// only, while SQL engines resolve declarative columns by JSON key — the same
// `WithFilter("started_at", …)` was correct on sqlite and silently matched
// nothing on memory. Resolution is now json-tag-first with the field-name
// fallback.

type tagRow struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"started_at"`
	State     string    `json:"state"`
}

func TestItemFieldByNameResolvesJSONTagFirst(t *testing.T) {
	t.Parallel()

	row := tagRow{ID: "r1", StartedAt: time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC), State: "open"}

	cases := []struct {
		name  string
		column string
		want  any
	}{
		{"json key snake_case", "started_at", row.StartedAt},
		{"json key case-insensitive", "STARTED_AT", row.StartedAt},
		{"go field name fallback", "StartedAt", row.StartedAt},
		{"go field name case-insensitive", "startedat", row.StartedAt},
		{"untagged field name", "state", "open"},
		{"missing column", "ended_at", nil},
	}

	for _, tc := range cases {
		got := metaengine.ItemFieldByName(row, tc.column)
		if got != tc.want {
			t.Errorf("%s: RowFieldByName(row, %q) = %v, want %v", tc.name, tc.column, got, tc.want)
		}
	}
}
