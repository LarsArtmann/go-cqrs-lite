package sqliteengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MapScanKeyValues implements metaengine.KeyScanBackend: a paged key+value
// read over the BASE meta_map table (never the planned table), in
// deterministic key order — the read primitive for planned-table backfill.
// cursor is the last key of the previous page (nil/"" for the first page).
func (e *sqliteEngine) MapScanKeyValues(
	ctx context.Context,
	collection string,
	cursor any,
	limit int,
) ([]any, []any, bool, error) {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	if limit <= 0 {
		limit = 500
	}

	var cursorArg any
	if cursor != nil {
		cursorArg = fmt.Sprint(cursor)
	}

	rows, err := e.db.QueryContext(
		ctx,
		`SELECT key, value FROM meta_map
		 WHERE collection = ? AND (? IS NULL OR key > ?)
		 ORDER BY key
		 LIMIT ?`,
		collection, cursorArg, cursorArg, limit,
	)
	if err != nil {
		return nil, nil, false, fmt.Errorf("sqliteengine.MapScanKeyValues: %w", err)
	}

	defer metaengine.DeferClose(rows)

	keys := make([]any, 0, limit)
	values := make([]any, 0, limit)

	for rows.Next() {
		var key string

		var raw string

		if err := rows.Scan(&key, &raw); err != nil {
			return nil, nil, false, fmt.Errorf("sqliteengine.MapScanKeyValues: scan: %w", err)
		}

		var val any

		if err := json.Unmarshal([]byte(raw), &val); err != nil {
			return nil, nil, false, fmt.Errorf("sqliteengine.MapScanKeyValues: unmarshal: %w", err)
		}

		keys = append(keys, key)
		values = append(values, val)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, false, fmt.Errorf("sqliteengine.MapScanKeyValues: rows: %w", err)
	}

	return keys, values, len(keys) == limit, nil
}

// EvolveLayoutPlan implements metaengine.LayoutPlanEvolver. It reconciles
// the planned table's physical columns with the plan using PRAGMA
// table_info: missing columns are added (ALTER TABLE ADD COLUMN). Idempotent
// — a matching schema applies nothing. The registered plan is replaced with
// the given plan (evolution intent is a schema change, so the conflict
// check from ApplyLayoutPlan does not apply).
//
// SQLite cannot ALTER COLUMN TYPE: a declared-type drift on an existing
// column fails loudly (naming the column and both types) instead of
// silently keeping the drifted affinity — recreate the table to retype.
func (e *sqliteEngine) EvolveLayoutPlan(
	ctx context.Context,
	plan metaengine.LayoutPlan,
) ([]string, error) {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	if _, exists := e.plans[plan.Collection]; !exists {
		if err := e.registerLayout(plan); err != nil {
			return nil, fmt.Errorf("sqliteengine.EvolveLayoutPlan: %w", err)
		}
	}

	have, err := sqliteTableColumns(ctx, e.db, plan.Table)
	if err != nil {
		return nil, fmt.Errorf("sqliteengine.EvolveLayoutPlan: %w", err)
	}

	applied := make([]string, 0)

	for _, c := range plan.Columns {
		got, exists := have[strings.ToLower(c.Name)]

		switch {
		case !exists:
			if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
				"ALTER TABLE %s ADD COLUMN %s %s",
				metaengine.QuoteIdent(plan.Table),
				metaengine.QuoteIdent(c.Name),
				c.Type,
			)); err != nil {
				return nil, fmt.Errorf("sqliteengine.EvolveLayoutPlan: add %s: %w", c.Name, err)
			}

			applied = append(applied, "add:"+c.Name)
		case !strings.EqualFold(got, c.Type):
			return nil, fmt.Errorf(
				"sqliteengine.EvolveLayoutPlan: column %s has declared type %s, want %s — "+
					"SQLite cannot ALTER COLUMN TYPE; recreate the table to retype",
				c.Name, got, c.Type)
		}
	}

	for _, idx := range plan.Indexes {
		if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
			metaengine.QuoteIdent(idx.Name),
			metaengine.QuoteIdent(plan.Table),
			metaengine.QuoteIdent(idx.Columns[0]),
		)); err != nil {
			return nil, fmt.Errorf("sqliteengine.EvolveLayoutPlan: index %s: %w", idx.Name, err)
		}
	}

	e.plans[plan.Collection] = plan

	return applied, nil
}

// sqliteTableColumns returns the table's columns as lower(name) → declared
// type, via PRAGMA table_info.
func sqliteTableColumns(ctx context.Context, db *sql.DB, table string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+metaengine.QuoteIdent(table)+")")
	if err != nil {
		return nil, fmt.Errorf("table_info %s: %w", table, err)
	}

	defer metaengine.DeferClose(rows)

	columns := make(map[string]string)

	for rows.Next() {
		var (
			cid       int
			name      string
			declType  string
			notNull   int
			dfltValue any
			pk        int
		)

		if err := rows.Scan(&cid, &name, &declType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("table_info %s: scan: %w", table, err)
		}

		columns[strings.ToLower(name)] = declType
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("table_info %s: rows: %w", table, err)
	}

	return columns, nil
}

// PlannedTables implements metaengine.PlannedTablesReporter: every registered
// planned collection with a live row count, in deterministic collection
// order. Row counts report -1 when the COUNT query fails (e.g. the table was
// dropped out-of-band) rather than failing the whole listing.
func (e *sqliteEngine) PlannedTables(ctx context.Context) ([]metaengine.PlannedTableInfo, error) {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	planList := make([]metaengine.LayoutPlan, 0, len(e.plans))
	for _, plan := range e.plans {
		planList = append(planList, plan)
	}

	sort.Slice(planList, func(i, j int) bool {
		return planList[i].Collection < planList[j].Collection
	})

	infos := make([]metaengine.PlannedTableInfo, 0, len(planList))

	for _, plan := range planList {
		info := metaengine.PlannedTableInfo{
			Collection: plan.Collection,
			Table:      plan.Table,
			Columns:    plan.ColumnNames(),
			Rows:       -1,
		}

		var n int64

		if err := e.db.QueryRowContext(
			ctx, "SELECT COUNT(*) FROM "+metaengine.QuoteIdent(plan.Table),
		).Scan(&n); err == nil {
			info.Rows = n
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// compile-time capability pins.
var (
	_ metaengine.KeyScanBackend        = (*sqliteEngine)(nil)
	_ metaengine.LayoutPlanEvolver     = (*sqliteEngine)(nil)
	_ metaengine.PlannedTablesReporter = (*sqliteEngine)(nil)
)
