package metaengine

import (
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrPlannedColumnTypeMismatch classifies filter, sort-cursor, or IN-member
// values whose Go type contradicts the planned column's declared type (for
// example a string filter on a BIGINT extracted column). It is classified as
// Rejection: the caller's query contradicts the registered LayoutPlan, so
// retrying cannot succeed — fix the filter value or the plan.
//
// Engines detect the mismatch at query-build time (before any SQL executes),
// so the failure is deterministic and engine-independent. The WRITE path
// keeps its fail-loud driver-level Infrastructure behavior: a document whose
// extracted value contradicts the column type is rejected by the storage
// engine when the row is written (pinned per engine, e.g.
// TestPgPlannedTable_MisTypedExtractFailsLoudly).
var ErrPlannedColumnTypeMismatch error = errorfamily.NewRejection(
	"metaengine.planned_column_type_mismatch",
	"value type contradicts the planned column type",
)

// PlannedColumnTypeCompatible reports whether val's Go type can bind to a
// planned column of the given declared type. Column types are the canonical
// names produced by BuildLayoutPlan and sqlTypeOf ("INTEGER", "BIGINT",
// "REAL", "DOUBLE", "TEXT") plus the engine dialect aliases; unknown or
// empty types are permissive so hand-built plans never get falsely rejected.
//
// Numeric Go values bind to numeric columns (INTEGER/BIGINT/REAL/DOUBLE/
// FLOAT); only Go strings bind to TEXT columns. Everything else (bool, nil,
// slices, maps) is a mismatch and the caller must surface
// ErrPlannedColumnTypeMismatch.
func PlannedColumnTypeCompatible(columnType string, val any) bool {
	switch strings.ToUpper(columnType) {
	case "INTEGER", "BIGINT", "REAL", "DOUBLE", "FLOAT":
		switch val.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			return true
		default:
			return false
		}
	case "TEXT":
		_, ok := val.(string)

		return ok
	default:
		return true
	}
}

// PlannedColumnType returns the declared type of the named column in the
// plan, and whether the column exists at all.
func PlannedColumnType(plan LayoutPlan, column string) (string, bool) {
	for _, c := range plan.Columns {
		if c.Name == column {
			return c.Type, true
		}
	}

	return "", false
}
