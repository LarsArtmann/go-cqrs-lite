package mysqlengine

import (
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// buildMapScanQuery renders the meta_map pushdown SELECT: the JSON-path
// filters (with their IN (...) form), the keyset cursor predicate and the
// ORDER BY with its key tiebreak. Both [(*mysqlEngine).PushdownMapScan] and
// [(*mysqlEngine).ExplainScanQuery] call it, so EXPLAIN describes the query
// that runs.
func (e *mysqlEngine) buildMapScanQuery(
	collection string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any, error) {
	if err := metaengine.ValidateQueryIdents(filters, sort); err != nil {
		return "", nil, fmt.Errorf("mysqlengine: %w", err)
	}

	var b strings.Builder

	args := []any{collection}

	b.WriteString("SELECT CAST(value AS CHAR), `key` FROM meta_map WHERE collection = ?")

	for _, f := range filters {
		e.appendMapFilter(&b, &args, f)
	}

	e.appendMapScanTail(&b, &args, sort, cursor)

	if limit > 0 {
		fmt.Fprintf(&b, ` LIMIT %d`, limit+1)
	}

	return b.String(), args, nil
}

// appendMapFilter writes " AND <expr> op ?" for one filter on meta_map; a
// FilterIn renders IN (?[, ...]) and an empty or non-list value renders
// [emptyInPredicate].
func (e *mysqlEngine) appendMapFilter(b *strings.Builder, args *[]any, f metaengine.FilterSpec) {
	if f.Op != metaengine.FilterIn {
		fmt.Fprintf(b, ` AND %s %s %s`,
			e.filterExpr(f.Column), string(f.Op), e.jsonParamPlaceholder())
		*args = append(*args, e.jsonFilterParam(f.Value))

		return
	}

	values, ok := f.Value.([]any)
	if !ok || len(values) == 0 {
		b.WriteString(" AND " + emptyInPredicate)

		return
	}

	placeholders := make([]string, len(values))
	for i, v := range values {
		placeholders[i] = e.jsonParamPlaceholder()
		*args = append(*args, e.jsonFilterParam(v))
	}

	fmt.Fprintf(b, ` AND %s IN (%s)`, e.filterExpr(f.Column), strings.Join(placeholders, ", "))
}

// appendMapScanTail writes the keyset cursor predicate (a compound
// [metaengine.SortKeyCursor] compares sort, sort =, key >) and the ORDER BY
// (sort expressions, then `key` ascending as the tiebreak).
func (e *mysqlEngine) appendMapScanTail(
	b *strings.Builder,
	args *[]any,
	sort *metaengine.SortSpec,
	cursor any,
) {
	if sort == nil {
		return
	}

	if cursor != nil {
		e.appendMapCursor(b, args, sort, cursor)
	}

	exprs := e.jsonSortExprs(sort.Column)
	for i := range exprs {
		if sort.Desc {
			exprs[i] += ` DESC`
		}
	}

	fmt.Fprintf(b, ` ORDER BY %s`, strings.Join(exprs, ", "))
	b.WriteString(", `key`")
}

// appendMapCursor writes the keyset predicate against the sort expressions
// the ORDER BY uses (see jsonCursorExpr); a compound cursor compares
// (sort op, sort =, key >) with the key tiebreak always ascending.
func (e *mysqlEngine) appendMapCursor(b *strings.Builder, args *[]any, sort *metaengine.SortSpec, cur any) {
	op := ">"
	if sort.Desc {
		op = "<"
	}

	skc, compound := cur.(metaengine.SortKeyCursor)

	sortExpr := e.jsonCursorExpr(sort.Column, cur)
	if compound {
		sortExpr = e.jsonCursorExpr(sort.Column, skc.Sort)
	}

	if !compound {
		fmt.Fprintf(b, ` AND %s %s %s`, sortExpr, op, e.jsonParamPlaceholder())
		*args = append(*args, e.jsonFilterParam(cur))

		return
	}

	// The compound predicate is parenthesized: ORDER BY must not see it open.
	fmt.Fprintf(b, ` AND (%s %s %s OR %s = %s AND `+"`key`"+` > ?)`,
		sortExpr, op, e.jsonParamPlaceholder(), sortExpr, e.jsonParamPlaceholder())

	sortParam := e.jsonFilterParam(skc.Sort)
	*args = append(*args, sortParam, sortParam, string(skc.Key))
}

// emptyInPredicate is the false predicate rendered for a FilterIn whose list
// is empty or not a list (ruling S041-02: such a filter matches no rows).
const emptyInPredicate = "1 = 0"

// rejectedExplain is what an explain surface returns instead of SQL when its
// input fails [metaengine.ValidateQueryIdents] or the planned-query builder;
// explains have no error return.
const rejectedExplain = "/* rejected: invalid filter or column */"
