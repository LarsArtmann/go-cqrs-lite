package metaengine

import (
	"encoding/json/v2"
	"fmt"
	"math"
	"strings"
)

// AppendKeysetCursorPredicate writes the keyset cursor predicate shared by the
// SQL engines' scan builders, handling both cursor shapes:
//
//   - legacy scalar cursor: `sortExpr op ?` (one bind arg)
//   - compound [SortKeyCursor]: `(sortExpr op ? OR sortExpr = ? AND keyExpr > ?)`
//     (three bind args: sort, sort, key)
//
// op flips with the sort direction; the key tiebreak comparison always
// ascends, mirroring the bytes.Compare tiebreak SortPaginate applies on the
// closure engines — without that symmetry a compound cursor issued by one
// page walk could not be consumed exactly-once by the next. sortExpr and
// keyExpr arrive fully rendered (quoted column idents for planned tables,
// json_extract expressions for standard tables); placeholder follows the
// dialect. Compound Sort values are normalized through
// normalizeCursorSortValue so JSON-round-tripped integral floats bind as
// int64.
func AppendKeysetCursorPredicate(
	b *strings.Builder,
	args *[]any,
	started *bool,
	sortExpr string,
	keyExpr string,
	desc bool,
	cursor any,
	placeholder func(int) string,
) {
	op := ">"
	if desc {
		op = "<"
	}

	beginFilterClause(b, started)

	skc, compound := cursor.(SortKeyCursor)
	if !compound {
		fmt.Fprintf(b, "%s %s %s", sortExpr, op, placeholder(len(*args)))

		*args = append(*args, cursor)

		return
	}

	fmt.Fprintf(b, "(%s %s %s OR %s = %s AND %s > %s)",
		sortExpr, op, placeholder(len(*args)),
		sortExpr, placeholder(len(*args)+1),
		keyExpr, placeholder(len(*args)+2))

	sortVal := normalizeCursorSortValue(skc.Sort)

	*args = append(*args, sortVal, sortVal, string(skc.Key))
}

// AppendKeysetOrder writes the ORDER BY tail for keyset scans: the sort
// expression in the requested direction, then the key expression ascending —
// the deterministic tiebreak the compound cursor predicate mirrors. Without
// the key tiebreak, rows sharing the sort value have no stable engine order
// and compound pagination cannot be exact-once.
func AppendKeysetOrder(b *strings.Builder, sortExpr, keyExpr string, desc bool) {
	b.WriteString(" ORDER BY ")
	b.WriteString(sortExpr)

	if desc {
		b.WriteString(" DESC")
	}

	b.WriteString(", ")
	b.WriteString(keyExpr)
}

// LastJSONRowCursor derives the compound continuation cursor from the last raw
// JSON row of a SQL scan: Sort is the row's sort-column value (JSON numbers
// decode as float64 — normalizeCursorSortValue handles integral ones at bind
// time), Key is the row's key column. Engines call it after trimming to the
// limit so the has-more probe row is never the cursor source. Returns nil for
// undecodable rows and rows missing the sort column; callers assign
// unconditionally.
func LastJSONRowCursor(rawValue []byte, column, key string) any {
	var row map[string]any
	if err := json.Unmarshal(rawValue, &row); err != nil {
		return nil
	}

	sortVal := ItemFieldByName(row, column)
	if sortVal == nil {
		return nil
	}

	return SortKeyCursor{Sort: sortVal, Key: []byte(key)}
}

// LastDecodedRowCursor is LastJSONRowCursor's counterpart for already-decoded
// scan rows (map[string]any from SQL engines' pushdown paths): Sort is the
// row's sort-column value, Key the row's key column as bytes. Returns nil for
// rows missing the sort column; callers assign unconditionally.
func LastDecodedRowCursor(row any, column string, key []byte) any {
	sortVal := ItemFieldByName(row, column)
	if sortVal == nil {
		return nil
	}

	return SortKeyCursor{Sort: sortVal, Key: key}
}

// normalizeCursorSortValue converts integral float64 cursor components back
// to int64. JSON round-trips (Cursor.Encode → ParseCursor) turn every number
// into float64; binding those against integer SQL columns risks driver-side
// type mismatch (e.g. PostgreSQL float8 vs int comparisons), while int64
// binds cleanly everywhere.
func normalizeCursorSortValue(v any) any {
	f, ok := v.(float64)
	if !ok || f != math.Trunc(f) || math.Abs(f) >= 1<<53 {
		return v
	}

	return int64(f)
}
