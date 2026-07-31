package metaengine

import (
	"fmt"
	"strings"
)

// appendStandardFilter writes a single filter clause for the standard
// (non-planned) path using json_extract. It handles both binary ops (=, <, etc.)
// and the special FilterIn operator which generates IN (?, ?, ...).
//
// The caller is responsible for the leading " AND " or " WHERE ".
func appendStandardFilter(b *strings.Builder, args *[]any, f FilterSpec) {
	path := jsonPath(f.Column)

	if f.Op == FilterIn {
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
// using direct column references. It handles both binary ops and FilterIn.
//
// The caller passes a pointer to started so the first clause gets " WHERE "
// and subsequent ones get " AND ".
func appendPlannedFilter(b *strings.Builder, args *[]any, f FilterSpec, started *bool) {
	if f.Op == FilterIn { //nolint:nestif // type assert + branch
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

		fmt.Fprintf(b, "%s IN (%s)", quoteIdent(f.Column), strings.Join(placeholders, ","))
	} else {
		if !*started {
			b.WriteString(" WHERE ")

			*started = true
		} else {
			b.WriteString(" AND ")
		}

		fmt.Fprintf(b, "%s %s ?", quoteIdent(f.Column), string(f.Op))
		*args = append(*args, f.Value)
	}
}
