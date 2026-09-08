package metaengine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// MaterializedViewsDoctorSection returns the "--- Materialized views ---"
// text block for Doctor() output: one line per operator-declared materialized
// view per engine, with the physical view name, aggregate shape, and live row
// count, plus an explicit "none" when no engine reports materialized views.
// Operators use it to verify their deployment-time accelerations are actually
// registered and being maintained.
func (s *Store) MaterializedViewsDoctorSection(_ context.Context) string {
	var b strings.Builder

	b.WriteString("\n--- Materialized views ---\n")

	s.mu.RLock()
	engines := s.engines
	s.mu.RUnlock()

	reported := false

	for _, eng := range engines {
		reporter, ok := eng.(MaterializedViewsReporter)
		if !ok {
			continue
		}

		for _, info := range reporter.MaterializedViews() {
			shape := string(info.Spec.Fn) + "(" + info.Spec.Column + ")"
			if info.Spec.Fn == MatViewCount {
				shape = "COUNT(*)"
			}

			if info.Spec.GroupBy != "" {
				shape += " BY " + info.Spec.GroupBy
			}

			rows := "N/A"
			if info.Rows >= 0 {
				rows = strconv.FormatInt(info.Rows, 10)
			}

			fmt.Fprintf(&b, "  %s: %s [%s] (rows=%s)\n",
				info.Spec.Collection, info.Name, shape, rows)

			if info.Spec.GroupBy != "" {
				fmt.Fprintf(&b, "    WARN: grouped views return silently wrong aggregates once a group is updated by a second transaction (turso-go <= v0.8.0-pre.8, verified 2026-09-07); scalar views are the safe shape\n")
			}

			reported = true
		}
	}

	if !reported {
		b.WriteString("  none\n")
	}

	return b.String()
}
