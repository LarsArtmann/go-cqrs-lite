package mysqlengine

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
// missing columns are added (existence-checked ADD COLUMN — plain syntax so
// the path works on Oracle MySQL, not just MariaDB), drifted types are
// retyped (MODIFY COLUMN). Idempotent — a matching schema applies nothing.
// The registered plan is replaced with the given plan (evolution intent is a
// schema change, so the conflict check from ApplyLayoutPlan does not apply).
func (e *mysqlEngine) EvolveLayoutPlan(ctx context.Context, plan metaengine.LayoutPlan) ([]string, error) {
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if _, exists := e.plans[plan.Collection]; !exists {
		if err := e.registerPlannedLayout(ctx, plan); err != nil {
			return nil, fmt.Errorf("mysqlengine.EvolveLayoutPlan: %w", err)
		}
	}

	applied := make([]string, 0)

	for _, c := range plan.Columns {
		want := strings.ToLower(mysqlPlannedColumn(c.Type))

		var got string

		err := e.db.QueryRowContext(
			ctx,
			`SELECT data_type FROM information_schema.columns
			 WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
			plan.Table, c.Name,
		).Scan(&got)

		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
				"ALTER TABLE %s ADD COLUMN %s %s",
				backtickIdent(plan.Table),
				backtickIdent(c.Name),
				mysqlPlannedColumn(c.Type),
			)); err != nil {
				return nil, fmt.Errorf("mysqlengine.EvolveLayoutPlan: add %s: %w", c.Name, err)
			}

			applied = append(applied, "add:"+c.Name)
		case err != nil:
			return nil, fmt.Errorf("mysqlengine.EvolveLayoutPlan: introspect %s: %w", c.Name, err)
		case strings.ToLower(got) != want:
			if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
				"ALTER TABLE %s MODIFY COLUMN %s %s",
				backtickIdent(plan.Table),
				backtickIdent(c.Name),
				mysqlPlannedColumn(c.Type),
			)); err != nil {
				return nil, fmt.Errorf("mysqlengine.EvolveLayoutPlan: retype %s: %w", c.Name, err)
			}

			applied = append(applied, "retype:"+c.Name)
		}
	}

	for _, idx := range plan.Indexes {
		if _, err := e.db.ExecContext(ctx, fmt.Sprintf(
			"CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
			backtickIdent(idx.Name),
			backtickIdent(plan.Table),
			backtickIdent(idx.Columns[0]),
		)); err != nil {
			return nil, fmt.Errorf("mysqlengine.EvolveLayoutPlan: index %s: %w", idx.Name, err)
		}
	}

	e.plans[plan.Collection] = plan

	return applied, nil
}
