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

// AppendPlannedCursor writes the keyset cursor predicate for planned-table
// scans: the WHERE/AND switch, then `col op <placeholder>` (op flips with
// sort direction), then appends the cursor argument. identQuote and
// placeholder follow the dialect (see [AppendPlannedFilter]).
func AppendPlannedCursor(
	b *strings.Builder,
	args *[]any,
	started *bool,
	sort *SortSpec,
	cursor any,
	identQuote func(string) string,
	placeholder func(int) string,
) {
	op := ">"
	if sort.Desc {
		op = "<"
	}

	beginFilterClause(b, started)

	fmt.Fprintf(b, "%s %s %s", identQuote(sort.Column), op, placeholder(len(*args)))

	*args = append(*args, cursor)
}

// AppendPlannedOrderLimit writes the ORDER BY / LIMIT tail shared by the SQL
// engines' planned-table scan builders: ORDER BY the quoted sort column
// (DESC when requested), then LIMIT limit+1 — the keyset HasMore probe. Both
// engines interpolate the integer limit directly; there is no bind parameter.
func AppendPlannedOrderLimit(
	b *strings.Builder,
	sort *SortSpec,
	limit int,
	identQuote func(string) string,
) {
	if sort != nil {
		fmt.Fprintf(b, " ORDER BY %s", identQuote(sort.Column))

		if sort.Desc {
			b.WriteString(" DESC")
		}
	}

	if limit > 0 {
		fmt.Fprintf(b, " LIMIT %d", limit+1)
	}
}
