package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// NewPlannedSQLiteEngine creates a SQLite engine with layout-planned tables
// for the given collections. Collections without a plan fall back to the
// standard meta_map table.
//
// As of ADR-0073 update, metaengine.Plan() auto-applies layouts for queries using
// FilterOnField/SortOnField — manual NewPlannedSQLiteEngine is only needed
// when you want explicit control over the metaengine.LayoutPlan (e.g., custom column types
// via metaengine.BuildLayoutPlanFromType[R]).
func NewPlannedSQLiteEngine(
	database *sql.DB,
	plans []metaengine.LayoutPlan,
) (metaengine.Engine, error) {
	eng, err := NewSQLiteEngine(database)
	if err != nil {
		return nil, err
	}

	sqlEng := eng.(*sqliteEngine)

	for _, plan := range plans {
		if err := sqlEng.registerLayout(plan); err != nil {
			return nil, fmt.Errorf("planned engine: create table %s: %w", plan.Table, err)
		}
	}

	return sqlEng, nil
}

// ApplyLayout implements metaengine.LayoutPlanner. It auto-generates a metaengine.LayoutPlan from
// the declared filter/sort field names and registers it on this engine. Called
// automatically by metaengine.Plan() for queries that use FilterOnField/SortOnField.
func (e *sqliteEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	if e.plans == nil {
		e.plans = make(map[string]metaengine.LayoutPlan)
	}

	if existing, exists := e.plans[collection]; exists {
		// Idempotent: same column set → no-op. Different columns → conflict.
		newPlan := metaengine.BuildLayoutPlan(collection, filterFields, sortFields)
		if !metaengine.PlansColumnCompatible(existing, newPlan) {
			return fmt.Errorf(
				"%w: collection %q already has columns %v, requested %v",
				metaengine.ErrLayoutConflict,
				collection,
				existing.ColumnNames(),
				newPlan.ColumnNames(),
			)
		}

		return nil // already planned with same columns
	}

	plan := metaengine.BuildLayoutPlan(collection, filterFields, sortFields)

	if err := e.registerLayout(plan); err != nil {
		return fmt.Errorf("apply layout %q: %w", collection, err)
	}

	return nil
}

// registerLayout creates the planned table + indexes and stores the plan.
func (e *sqliteEngine) registerLayout(plan metaengine.LayoutPlan) error {
	if _, err := e.db.ExecContext(context.Background(), plan.DDL()); err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	if e.plans == nil {
		e.plans = make(map[string]metaengine.LayoutPlan)
	}

	e.plans[plan.Collection] = plan

	return nil
}

// ApplyLayoutPlan implements metaengine.LayoutPlanApplier. It registers a
// full LayoutPlan on the engine post-construction, enabling runtime layout
// registration without recreating the engine. Mirrors DuckDB's ApplyLayoutPlan.
func (e *sqliteEngine) ApplyLayoutPlan(plan metaengine.LayoutPlan) error {
	if e.plans == nil {
		e.plans = make(map[string]metaengine.LayoutPlan)
	}

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

	if err := e.registerLayout(plan); err != nil {
		return fmt.Errorf("sqliteengine.ApplyLayoutPlan: create table %s: %w", plan.Table, err)
	}

	return nil
}
