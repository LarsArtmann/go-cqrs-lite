package pgengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Planned tables (D1): extracted-column tables per collection, replacing the
// JSONB-everything meta_map path for collections with a registered
// metaengine.LayoutPlan. Mirrors metaengine/sqliteengine/planned.go — the
// PG dialect uses JSONB for the value column, $N placeholders, and
// ON CONFLICT upserts.
//
// Scope note (2026-08-30): this slice registers plans and routes
// MapSet/MapGet/MapDelete. Counter/graph/aggregate and full pushdown scan
// routing remain meta_map paths until D3 lands.

// pgPlannedColumn maps a plan's SQLite-ish inferred type to a PG type.
// REAL→DOUBLE PRECISION keeps float64 lossless; INTEGER and TEXT pass through.
func pgPlannedColumn(sqliteType string) string {
	switch strings.ToUpper(sqliteType) {
	case "REAL":
		return "DOUBLE PRECISION"
	case "INTEGER":
		return "BIGINT"
	default:
		return "TEXT"
	}
}

// pgDDL renders the CREATE TABLE + CREATE INDEX statements for a plan in the
// Postgres dialect.
func pgDDL(plan metaengine.LayoutPlan) string {
	var b strings.Builder

	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (\n", metaengine.QuoteIdent(plan.Table))
	b.WriteString("  key TEXT PRIMARY KEY,\n")
	b.WriteString("  value JSONB NOT NULL")

	for _, c := range plan.Columns {
		fmt.Fprintf(&b, ",\n  %s %s", metaengine.QuoteIdent(c.Name), pgPlannedColumn(c.Type))
	}

	b.WriteString("\n);")

	for _, idx := range plan.Indexes {
		fmt.Fprintf(&b, "\nCREATE INDEX IF NOT EXISTS %s ON %s(%s);",
			metaengine.QuoteIdent(idx.Name), metaengine.QuoteIdent(plan.Table),
			metaengine.QuoteIdent(idx.Columns[0]))
	}

	return b.String()
}

// planFor returns the registered plan for a collection, if any.
func (e *pgEngine) planFor(col string) (metaengine.LayoutPlan, bool) {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	plan, ok := e.plans[col]

	return plan, ok
}

// registerPlannedLayout creates the planned table + indexes and stores the
// plan. Called with layoutMu held.
func (e *pgEngine) registerPlannedLayout(plan metaengine.LayoutPlan) error {
	if _, err := e.db.ExecContext(context.Background(), pgDDL(plan)); err != nil {
		return fmt.Errorf("pgengine.registerPlannedLayout: %w", err)
	}

	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	if e.plans == nil {
		e.plans = make(map[string]metaengine.LayoutPlan)
	}

	e.plans[plan.Collection] = plan

	return nil
}

// art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
// ApplyLayoutPlan implements metaengine.LayoutPlanApplier: registers a full
// LayoutPlan (with reflection-derived column types) post-construction and
// creates the planned table. Conflicting re-registrations are rejected.
func (e *pgEngine) ApplyLayoutPlan(plan metaengine.LayoutPlan) error {
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if existing, exists := e.plans[plan.Collection]; exists {
		if !metaengine.PlansColumnCompatible(existing, plan) {
			return fmt.Errorf(
				"%w: collection %q already has columns %v, requested %v",
				metaengine.ErrLayoutConflict,
				plan.Collection,
				existing.ColumnNames(),
				plan.ColumnNames(),
			)
		}

		return nil
	}

	if err := e.registerPlannedLayout(plan); err != nil {
		return fmt.Errorf("pgengine.ApplyLayoutPlan: %w", err)
	}

	return nil
}

// mapSetPlanned upserts a key-value pair with extracted columns.
func (e *pgEngine) mapSetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
	value any,
) error {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	if err := execPlannedUpsert(ctx, e.conn(), plan, fmt.Sprint(key), value); err != nil {
		return fmt.Errorf("pgengine.mapSetPlanned: %w", err)
	}

	return nil
}

// execPlannedUpsert writes the value + re-extracted columns to the planned
// table on the given executor (e.conn() for normal paths, a transaction for
// MapUpdate). Shared by MapSet and the MapUpdate read-modify-write so the
// extracted columns stay consistent with the JSONB value on every write.
func execPlannedUpsert(
	ctx context.Context,
	q metaengine.SQLExec,
	plan metaengine.LayoutPlan,
	keyStr string,
	value any,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	extracted := metaengine.ExtractFields(value, plan.Columns)

	cols := make([]string, 0, 2+len(plan.Columns))
	placeholders := make([]string, 0, 2+len(plan.Columns))
	args := make([]any, 0, 2+len(plan.Columns))

	cols = append(cols, "key", "value")
	placeholders = append(placeholders, "$1", "$2::jsonb")
	args = append(args, keyStr, string(data))

	for i, c := range plan.Columns {
		cols = append(cols, metaengine.QuoteIdent(c.Name))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+3))
		args = append(args, extracted[c.Name])
	}

	updates := make([]string, 0, 1+len(plan.Columns))
	updates = append(updates, "value = excluded.value")
	for _, c := range plan.Columns {
		updates = append(updates,
			fmt.Sprintf("%s = excluded.%s", metaengine.QuoteIdent(c.Name), metaengine.QuoteIdent(c.Name)))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (key) DO UPDATE SET %s",
		metaengine.QuoteIdent(plan.Table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updates, ", "),
	)

	if _, err := q.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

// mapGetPlanned reads one row from the planned table.
func (e *pgEngine) mapGetPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
) (any, bool, error) {
	var raw []byte

	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	err := e.conn().QueryRowContext(
		ctx,
		fmt.Sprintf("SELECT value::text FROM %s WHERE key = $1",
			metaengine.QuoteIdent(plan.Table)),
		fmt.Sprint(key),
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("pgengine.mapGetPlanned: %w", err)
	}

	var val any
	if err := json.Unmarshal(raw, &val); err != nil {
		return nil, false, fmt.Errorf("pgengine.mapGetPlanned: unmarshal: %w", err)
	}

	return val, true, nil
}

// mapDeletePlanned removes one row from the planned table.
func (e *pgEngine) mapDeletePlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	key any,
) error {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	_, err := e.conn().ExecContext(
		ctx,
		fmt.Sprintf("DELETE FROM %s WHERE key = $1", metaengine.QuoteIdent(plan.Table)),
		fmt.Sprint(key),
	)
	if err != nil {
		return fmt.Errorf("pgengine.mapDeletePlanned: %w", err)
	}

	return nil
}
