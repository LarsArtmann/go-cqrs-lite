package pgengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// EvolveLayoutPlan implements metaengine.LayoutPlanEvolver. It reconciles the
// planned table's physical columns with the plan using information_schema:
// missing columns are added (ADD COLUMN IF NOT EXISTS), drifted types are
// retyped (ALTER COLUMN TYPE). Idempotent — a matching schema applies nothing.
// The registered plan is replaced with the given plan (evolution intent is a
// schema change, so the conflict check from ApplyLayoutPlan does not apply).
func (e *pgEngine) EvolveLayoutPlan(ctx context.Context, plan metaengine.LayoutPlan) ([]string, error) {
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if _, exists := e.plans[plan.Collection]; !exists {
		if err := e.registerPlannedLayout(ctx, plan); err != nil {
			return nil, fmt.Errorf("pgengine.EvolveLayoutPlan: %w", err)
		}
	}

	applied := make([]string, 0)

	for _, c := range plan.Columns {
		want := strings.ToLower(pgPlannedColumn(c.Type))

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
				"ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s",
				metaengine.QuoteIdent(plan.Table),
				metaengine.QuoteIdent(c.Name),
				pgPlannedColumn(c.Type),
			)); err != nil {
				return nil, fmt.Errorf("pgengine.EvolveLayoutPlan: add %s: %w", c.Name, err)
			}

			applied = append(applied, "add:"+c.Name)
		case err != nil:
			return nil, fmt.Errorf("pgengine.EvolveLayoutPlan: introspect %s: %w", c.Name, err)
		case strings.ToLower(got) != want:
			// USING cast: text→numeric conversions need an explicit expression
			// (SQLSTATE 42804 otherwise). A value the target type cannot
			// represent fails the ALTER loudly — the no-data-loss default.
			if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
				"ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s::%s",
				metaengine.QuoteIdent(plan.Table),
				metaengine.QuoteIdent(c.Name),
				pgPlannedColumn(c.Type),
				metaengine.QuoteIdent(c.Name),
				pgPlannedColumn(c.Type),
			)); err != nil {
				return nil, fmt.Errorf("pgengine.EvolveLayoutPlan: retype %s: %w", c.Name, err)
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
			return nil, fmt.Errorf("pgengine.EvolveLayoutPlan: index %s: %w", idx.Name, err)
		}
	}

	e.plans[plan.Collection] = plan

	return applied, nil
}
