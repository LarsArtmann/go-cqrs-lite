package sqliteengine

// TrimGroupedCap: the sqliteengine side of per-group caps (ADR-0164). One
// window-function statement trims every group in the collection at once —
// the pushdown behind metaengine.CapGroupTrimmer.

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// capGroupIdent reports whether name is a bare identifier, safe to embed in a
// JSON path or quote as a column. The store validates at declaration time;
// this is the engine-side defensive re-check (TrimGroupedCap is only called
// by the store, but the engine must not be able to build broken SQL).
func capGroupIdent(name string) bool {
	if name == "" {
		return false
	}

	for i := 0; i < len(name); i++ {
		c := name[i]

		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
		case c >= '0' && c <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}

	return true
}

// capColumnExpr renders the SQL expression for one declared column: the
// planned table's REAL column when the layout extracted it, the JSON
// document's member otherwise (every sqliteengine table — meta_map and
// planned — carries the full value document).
func capColumnExpr(plan metaengine.LayoutPlan, col string) string {
	for _, name := range plan.ColumnNames() {
		if strings.EqualFold(name, col) {
			return metaengine.QuoteIdent(name)
		}
	}

	return jsonMemberExpr(col)
}

// jsonMemberExpr renders the json_extract member access for a column of the
// stored value document. col must already satisfy capGroupIdent; jsonPath adds
// the same quote escaping every other json_extract site in this engine uses.
func jsonMemberExpr(col string) string {
	return fmt.Sprintf("json_extract(value, '%s')", jsonPath(col))
}

// TrimGroupedCap implements [metaengine.CapGroupTrimmer]: it deletes the rows
// beyond the newest perGroupRows of each distinct groupCol value (ordered by
// orderBy descending) in one statement and returns the deleted keys.
//
// Cell-versioned engines return [metaengine.ErrCapGroupPushdownUnsupported]:
// a bulk DELETE would bypass meta_cell_versions, so the store trims through
// its Go path, which deletes per row via MapDeleteAt.
func (e *sqliteEngine) TrimGroupedCap(
	ctx context.Context,
	collection, groupCol, orderBy string,
	perGroupRows int,
) ([]any, error) {
	if e.versioning {
		return nil, fmt.Errorf(
			"sqliteengine.TrimGroupedCap: %s: %w",
			collection, metaengine.ErrCapGroupPushdownUnsupported,
		)
	}

	if !capGroupIdent(groupCol) || !capGroupIdent(orderBy) {
		return nil, fmt.Errorf(
			"sqliteengine.TrimGroupedCap: %s: %w: group %q order %q",
			collection, metaengine.ErrCapGroupPushdownUnsupported, groupCol, orderBy,
		)
	}

	table := "meta_map"

	where := " WHERE collection = ?"

	args := make([]any, 0, 3)

	args = append(args, collection)

	groupExpr := jsonMemberExpr(groupCol)
	orderExpr := jsonMemberExpr(orderBy)

	if plan, ok := e.plans[collection]; ok {
		table = metaengine.QuoteIdent(plan.Table)
		where = ""
		args = args[:0]
		groupExpr = capColumnExpr(plan, groupCol)
		orderExpr = capColumnExpr(plan, orderBy)
	}

	query := fmt.Sprintf(
		`DELETE FROM %s WHERE rowid IN (
			SELECT rowid FROM (
				SELECT rowid, ROW_NUMBER() OVER (
					PARTITION BY %s ORDER BY %s DESC
				) AS rn
				FROM %s%s
			) WHERE rn > ?
		) RETURNING key`,
		table, groupExpr, orderExpr, table, where,
	)

	args = append(args, perGroupRows)

	rows, err := e.xd(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.TrimGroupedCap: %s: %w", collection, err)
	}
	defer func() { _ = rows.Close() }()

	keys := make([]any, 0, 8)

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("sqliteengine.TrimGroupedCap: %s: scan: %w", collection, err)
		}

		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqliteengine.TrimGroupedCap: %s: %w", collection, err)
	}

	return keys, nil
}
