package pgengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// PlannedTables implements metaengine.PlannedTablesReporter: every registered
// planned collection with a live row count, in deterministic collection
// order. Row counts report -1 when the COUNT query fails (e.g. the table was
// dropped out-of-band) rather than failing the whole listing.
func (e *pgEngine) PlannedTables(ctx context.Context) ([]metaengine.PlannedTableInfo, error) {
	e.layoutMu.Lock()
	planList := make([]metaengine.LayoutPlan, 0, len(e.plans))
	for _, plan := range e.plans {
		planList = append(planList, plan)
	}
	e.layoutMu.Unlock()

	return metaengine.ListPlannedTables(ctx, e.db, planList), nil
}
