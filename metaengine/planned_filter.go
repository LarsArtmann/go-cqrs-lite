package metaengine

import (
	"fmt"
	"strings"
)

// QuestionPlaceholders renders ? placeholders — the bind style of the SQLite
// and MySQL engines.
func QuestionPlaceholders(int) string { return "?" }

// DollarPlaceholders renders $N placeholders (1-based) — the bind style of
// the Postgres engine; n is the 0-based count of already-appended args.
func DollarPlaceholders(n int) string { return fmt.Sprintf("$%d", n+1) }

// AppendPlannedFilter writes one planned-table filter clause: the FilterIn
// branch expands the value list into placeholders and renders an IN (...);
// every other operator renders a binary comparison. The WHERE/AND switch is
// shared (first clause " WHERE ", the rest " AND "). It is the shared core of
// the SQL engines' appendPlannedFilter builders; identQuote quotes the column
// name (backticks on MySQL, [QuoteIdent] elsewhere), placeholder renders
// each bind position, and inSeparator joins IN-list placeholders (", " on
// pg/mysql, "," on sqlite — the wire strings each dialect historically
// emitted; keep them byte-identical).
func AppendPlannedFilter(
	b *strings.Builder,
	args *[]any,
	f FilterSpec,
	started *bool,
	identQuote func(string) string,
	placeholder func(int) string,
	inSeparator string,
) {
	values, _ := f.Value.([]any)

	if f.Op == FilterIn {
		if len(values) == 0 {
			return
		}

		beginFilterClause(b, started)

		placeholders := make([]string, len(values))
		for i, v := range values {
			placeholders[i] = placeholder(len(*args))

			*args = append(*args, v)
		}

		fmt.Fprintf(b, "%s IN (%s)", identQuote(f.Column), strings.Join(placeholders, inSeparator))

		return
	}

	beginFilterClause(b, started)

	fmt.Fprintf(b, "%s %s %s", identQuote(f.Column), string(f.Op), placeholder(len(*args)))

	*args = append(*args, f.Value)
}

// beginFilterClause opens the WHERE clause or extends it with AND.
func beginFilterClause(b *strings.Builder, started *bool) {
	if !*started {
		b.WriteString(" WHERE ")

		*started = true
	} else {
		b.WriteString(" AND ")
	}
}
