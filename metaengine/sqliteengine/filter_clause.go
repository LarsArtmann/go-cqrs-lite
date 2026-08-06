package sqliteengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"fmt"
	"strings"
)

// appendStandardFilter writes a single filter clause for the standard
// (non-planned) path using json_extract. It handles both binary ops (=, <, etc.)
// and the special metaengine.FilterIn operator which generates IN (?, ?, ...).
//
// The caller is responsible for the leading " AND " or " WHERE ".
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
// using direct column references. It handles both binary ops and metaengine.FilterIn.
//
// The caller passes a pointer to started so the first clause gets " WHERE "
// and subsequent ones get " AND ".
func appendPlannedFilter(b *strings.Builder, args *[]any, f metaengine.FilterSpec, started *bool) {
	if f.Op == metaengine.FilterIn {
		values, ok := f.Value.([]any)
		if !ok || len(values) == 0 {
			return
		}

		if !*started {
			b.WriteString(" WHERE ")

			*started = true
		} else {
			b.WriteString(" AND ")
		}

		placeholders := make([]string, len(values))
		for i, v := range values {
			placeholders[i] = "?"

			*args = append(*args, v)
		}

		fmt.Fprintf(b, "%s IN (%s)", metaengine.QuoteIdent(f.Column), strings.Join(placeholders, ","))
	} else {
		if !*started {
			b.WriteString(" WHERE ")

			*started = true
		} else {
			b.WriteString(" AND ")
		}

		fmt.Fprintf(b, "%s %s ?", metaengine.QuoteIdent(f.Column), string(f.Op))
		*args = append(*args, f.Value)
	}
}
