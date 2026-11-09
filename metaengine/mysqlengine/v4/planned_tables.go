package mysqlengine

import (
	"context"
	"sort"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// PlannedTables implements metaengine.PlannedTablesReporter: every registered
// planned collection with a live row count, in deterministic collection
// order. Row counts report -1 when the COUNT query fails (e.g. the table was
// dropped out-of-band) rather than failing the whole listing.
func (e *mysqlEngine) PlannedTables(ctx context.Context) ([]metaengine.PlannedTableInfo, error) {
	e.layoutMu.Lock()
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	planList := make([]metaengine.LayoutPlan, 0, len(e.plans))
	for _, plan := range e.plans {
		planList = append(planList, plan)
	}
	e.layoutMu.Unlock()

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
			ctx, "SELECT COUNT(*) FROM "+backtickIdent(plan.Table),
		).Scan(&n); err == nil {
			info.Rows = n
		}

		infos = append(infos, info)
	}

	return infos, nil
}
