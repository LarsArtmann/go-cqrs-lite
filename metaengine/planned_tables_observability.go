package metaengine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// PlannedTableInfo describes one registered planned table for observability:
// the collection it serves, the physical table name, the extracted columns,
// and the live row count (-1 when the count could not be read).
type PlannedTableInfo struct {
	Collection string
	Table      string
	Columns    []string
	Rows       int64
}

// PlannedTablesReporter is an optional capability for SQL engines with
// planned tables (the LayoutPlanApplier family): it lists every registered
// planned collection with a live row count. Store.Doctor uses it to surface
// planned-table state — which collections have native extracted-column
// tables, and how much data they hold — so operators can see routing-relevant
// storage layout without querying the catalog by hand.
type PlannedTablesReporter interface {
	PlannedTables(ctx context.Context) ([]PlannedTableInfo, error)
}

// PlannedTablesDoctorSection returns the "--- Planned tables ---" text block
// for Doctor() output: one line per registered planned table per engine,
// with row counts, plus an explicit "none" when no engine reports planned
// tables.
func (s *Store) PlannedTablesDoctorSection(ctx context.Context) string {
	var b strings.Builder

	b.WriteString("\n--- Planned tables ---\n")

	s.mu.RLock()
	engines := s.engines
	s.mu.RUnlock()

	reported := false

	for _, eng := range engines {
		reporter, ok := eng.(PlannedTablesReporter)
		if !ok {
			continue
		}

		infos, err := reporter.PlannedTables(ctx)
		if err != nil {
			fmt.Fprintf(&b, "  %s: ERROR: %v\n", eng.Profile().Name, err)
			reported = true

			continue
		}

		for _, info := range infos {
			rows := "N/A"
			if info.Rows >= 0 {
				rows = strconv.FormatInt(info.Rows, 10)
			}

			fmt.Fprintf(&b, "  %s: %s (rows=%s, columns=%v)\n",
				info.Collection, info.Table, rows, info.Columns)
			reported = true
		}
	}

	if !reported {
		b.WriteString("  none\n")
	}

	return b.String()
}
