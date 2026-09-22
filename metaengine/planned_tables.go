package metaengine

import (
	"context"
	"sort"
)

// ListPlannedTables turns a snapshot of registered layout plans into
// PlannedTableInfos: deterministic collection order, one live row count per
// table via COUNT(*). Row counts report -1 when the COUNT query fails (e.g.
// the table was dropped out-of-band) rather than failing the whole listing.
// It is the shared body of the SQL engines' PlannedTablesReporter
// implementations; the caller owns snapshotting (and locking) its plans map,
// because lock mode (RLock vs Lock) is an engine decision.
func ListPlannedTables(ctx context.Context, q SQLExec, plans []LayoutPlan) []PlannedTableInfo {
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Collection < plans[j].Collection
	})

	infos := make([]PlannedTableInfo, 0, len(plans))

	for _, plan := range plans {
		info := PlannedTableInfo{
			Collection: plan.Collection,
			Table:      plan.Table,
			Columns:    plan.ColumnNames(),
			Rows:       -1,
		}

		var n int64

		if err := q.QueryRowContext(
			ctx, "SELECT COUNT(*) FROM "+QuoteIdent(plan.Table),
		).Scan(&n); err == nil {
			info.Rows = n
		}

		infos = append(infos, info)
	}

	return infos
}
