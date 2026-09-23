package sqliteengine

import (
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// appendStandardFilter writes a single filter clause for the standard
// (non-planned) path using json_extract. It handles both binary ops (=, <, etc.)
// and the special metaengine.FilterIn operator which generates IN (?, ?, ...).
//
// The caller is responsible for the leading " AND " or " WHERE ".
//
// Filter type-coercion: SQLite json_extract returns native types (INTEGER, REAL,
// TEXT), so numeric comparisons work without explicit CAST. This differs from
// DuckDB (CAST AS DOUBLE) and Postgres (::float8) — see the equivalent filter
// builders in those packages for their coercion strategies.
// art-dupl:accept cross-module SQL builder pattern — separate go.mod
func appendStandardFilter(b *strings.Builder, args *[]any, f metaengine.FilterSpec) {
	path := jsonPath(f.Column)

	if f.Op == metaengine.FilterIn {
		values, ok := f.Value.([]any)
		if !ok || len(values) == 0 {
			return
		}

		placeholders := make([]string, len(values))
		for i, v := range values {
			placeholders[i] = "?"

			*args = append(*args, v)
		}

		fmt.Fprintf(b,

			` AND json_extract(value, '%s') IN (%s)`,
			path,
			strings.Join(placeholders, ","))
	} else {
		fmt.Fprintf(b, ` AND json_extract(value, '%s') %s ?`, path, string(f.Op))
		*args = append(*args, f.Value)
	}
}

// appendPlannedFilter writes a single filter clause for the planned-table path
// using direct column references (QuoteIdent) and ? placeholders; the clause
// mechanics live in metaengine.AppendPlannedFilter.
func appendPlannedFilter(b *strings.Builder, args *[]any, f metaengine.FilterSpec, started *bool) {
	metaengine.AppendPlannedFilter(
		b,
		args,
		f,
		started,
		metaengine.QuoteIdent,
		metaengine.QuestionPlaceholders,
		",", // sqlite's historical space-free IN-list rendering
	)
}
