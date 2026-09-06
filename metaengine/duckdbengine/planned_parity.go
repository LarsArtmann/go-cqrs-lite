package duckdbengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sort"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MapScanKeyValues implements metaengine.KeyScanBackend: a paged key+value
// read over the BASE meta_map table (never the planned table), in
// deterministic key order — the read primitive for planned-table backfill.
// cursor is the last key of the previous page (nil/"" for the first page).
func (e *duckdbEngine) MapScanKeyValues(
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

	rows, err := e.conn().QueryContext(
		ctx,
		`SELECT key, value FROM meta_map
		 WHERE collection = $1 AND ($2::VARCHAR IS NULL OR key > $2)
		 ORDER BY key
		 LIMIT $3`,
		collection, cursorArg, limit,
	)
	if err != nil {
		return nil, nil, false, fmt.Errorf("duckdbengine.MapScanKeyValues: %w", err)
	}

	defer metaengine.DeferClose(rows)

	keys := make([]any, 0, limit)
	values := make([]any, 0, limit)

	for rows.Next() {
		var key string

		var raw string

		if err := rows.Scan(&key, &raw); err != nil {
			return nil, nil, false, fmt.Errorf("duckdbengine.MapScanKeyValues: scan: %w", err)
		}

		var val any

		if err := json.Unmarshal([]byte(raw), &val); err != nil {
			return nil, nil, false, fmt.Errorf("duckdbengine.MapScanKeyValues: unmarshal: %w", err)
		}

		keys = append(keys, key)
		values = append(values, val)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, false, fmt.Errorf("duckdbengine.MapScanKeyValues: rows: %w", err)
	}

	return keys, values, len(keys) == limit, nil
}

// EvolveLayoutPlan implements metaengine.LayoutPlanEvolver. It reconciles
// the planned table's physical columns with the plan using
// information_schema: missing columns are added (ADD COLUMN), drifted types
// are retyped (ALTER COLUMN TYPE). Idempotent — a matching schema applies
// nothing. The registered plan is replaced with the given plan (evolution
// intent is a schema change, so the conflict check from ApplyLayoutPlan
// does not apply).
func (e *duckdbEngine) EvolveLayoutPlan(
	ctx context.Context,
	plan metaengine.LayoutPlan,
) ([]string, error) {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if _, exists := e.plans[plan.Collection]; !exists {
		if err := e.applyLayoutPlanLocked(plan); err != nil {
			return nil, fmt.Errorf("duckdbengine.EvolveLayoutPlan: %w", err)
		}
	}

	applied := make([]string, 0)

	for _, c := range plan.Columns {
		want := strings.ToUpper(c.Type)

		var got string

		err := e.db.QueryRowContext(
			ctx,
			`SELECT data_type FROM information_schema.columns
			 WHERE table_name = $1 AND column_name = $2`,
			plan.Table, c.Name,
		).Scan(&got)

		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
				"ALTER TABLE %s ADD COLUMN %s %s",
				metaengine.QuoteIdent(plan.Table),
				metaengine.QuoteIdent(c.Name),
				c.Type,
			)); err != nil {
				return nil, fmt.Errorf("duckdbengine.EvolveLayoutPlan: add %s: %w", c.Name, err)
			}

			applied = append(applied, "add:"+c.Name)
		case err != nil:
			return nil, fmt.Errorf("duckdbengine.EvolveLayoutPlan: introspect %s: %w", c.Name, err)
		case !strings.EqualFold(got, want):
			if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
				"ALTER TABLE %s ALTER COLUMN %s TYPE %s",
				metaengine.QuoteIdent(plan.Table),
				metaengine.QuoteIdent(c.Name),
				c.Type,
			)); err != nil {
				return nil, fmt.Errorf("duckdbengine.EvolveLayoutPlan: retype %s: %w", c.Name, err)
			}

			applied = append(applied, "retype:"+c.Name)
		}
	}

	for _, idx := range plan.Indexes {
		if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
			metaengine.QuoteIdent(idx.Name),
			metaengine.QuoteIdent(plan.Table),
			metaengine.QuoteIdent(idx.Columns[0]),
		)); err != nil {
			return nil, fmt.Errorf("duckdbengine.EvolveLayoutPlan: index %s: %w", idx.Name, err)
		}
	}

	e.plans[plan.Collection] = plan

	return applied, nil
}

// PlannedTables implements metaengine.PlannedTablesReporter: every registered
// planned collection with a live row count, in deterministic collection
// order. Row counts report -1 when the COUNT query fails (e.g. the table was
// dropped out-of-band) rather than failing the whole listing.
func (e *duckdbEngine) PlannedTables(ctx context.Context) ([]metaengine.PlannedTableInfo, error) {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	e.layoutMu.RLock()
	planList := make([]metaengine.LayoutPlan, 0, len(e.plans))
	for _, plan := range e.plans {
		planList = append(planList, plan)
	}
	e.layoutMu.RUnlock()

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
	_ metaengine.KeyScanBackend        = (*duckdbEngine)(nil)
	_ metaengine.LayoutPlanEvolver     = (*duckdbEngine)(nil)
	_ metaengine.PlannedTablesReporter = (*duckdbEngine)(nil)
)
