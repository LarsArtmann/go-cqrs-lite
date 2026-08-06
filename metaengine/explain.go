package metaengine

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// ExplainOptions controls what EXPLAIN returns.
type ExplainOptions struct {
	Filters []FilterSpec
	Sort    *SortSpec
	Cursor  any
	Limit   int
}

// ExplainableScan is an optional capability for engines that can show the SQL
// query that would execute for a scan without running it. External engine
// packages (duckdbengine, pgengine) implement this to support
// TypedReader.Explain. The internal sqliteEngine uses an unexported method
// checked separately.
type ExplainableScan interface {
	ExplainScanQuery(ctx context.Context, collection string, opts ExplainOptions) (string, []any)
}

// Explain returns the SQL that would execute for a scan query, without running
// it. Useful for debugging pushdown, verifying index usage, and understanding
// query plans.
//
//	reader := metaengine.NewReader[V](store, "find_user")
//	sql, args := reader.Explain(ctx,
//	    metaengine.WithFilter("status", metaengine.FilterEq, "open"),
//	)
func (r *TypedReader[V]) Explain(
	ctx context.Context,
	opts ...ScanOption,
) (string, []any) {
	cfg := scanConfig{limit: 100}

	for _, opt := range opts {
		opt(&cfg)
	}

	eng, ok := r.store.collectionEngine(r.collection)
	if !ok {
		return "-- no engine for collection " + r.collection, nil
	}

	// Exported interface: external engines (duckdb, pg) that can't implement
	// the unexported scanConfig-based method.
	if es, ok := eng.(ExplainableScan); ok {
		filters := cfg.filters

		for _, rg := range cfg.ranges {
			filters = append(
				filters,
				FilterSpec{Column: rg.Column, Op: FilterGe, Value: rg.Low},
				FilterSpec{Column: rg.Column, Op: FilterLe, Value: rg.High},
			)
		}

		return es.ExplainScanQuery(ctx, r.collection, ExplainOptions{
			Filters: filters,
			Sort:    cfg.sort,
			Cursor:  cfg.cursor,
			Limit:   cfg.limit,
		})
	}

	return "-- EXPLAIN not supported by engine " + eng.Profile().Name, nil
}

// ─── Store-level observability (O3 + O4) ───

// ExplainPlan returns a human-readable explanation of the full query plan.
// It shows each engine's capabilities, each query's assignment (engine, ADT,
// complexity, cost), and any diagnostics.
//
// This is the primary tool for understanding WHY the planner assigned each
// query to a particular engine. Use it during development to verify
// FilterOnField/SortOnField declarations produce pushdown scans.
func (s *Store) ExplainPlan() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var b strings.Builder

	b.WriteString("=== Metaengine Plan ===\n\n--- Engines ---\n")

	for _, eng := range s.engines {
		p := eng.Profile()

		if rc := p.ReadCosts; rc.NsPerPointLookup > 0 || rc.NsPerFilteredScan > 0 ||
			rc.NsPerAggregate > 0 || rc.NsPerScan > 0 {
			fmt.Fprintf(
				&b,
				"  %s (point=%.0fns, scan=%.0fns/row, push=%.0fns/row, agg=%.0fns/row, write=%.0fns/op)",
				p.Name,
				p.NsForRead(ReadPointLookup),
				p.NsForRead(ReadScan),
				p.NsForRead(ReadFilteredScan),
				p.NsForRead(ReadAggregate),
				p.WriteNsPerOp(),
			)
		} else {
			fmt.Fprintf(&b, "  %s (read=%.0fns/op, write=%.0fns/op)",
				p.Name, p.ReadNsPerOp(), p.WriteNsPerOp())
		}
		if p.IsReplicated() {
			fmt.Fprintf(&b, " replication=%s, lag=%s, rtt=%s",
				p.Replication, p.EffectiveReplicationLag(), p.EffectiveNetworkRTT())
		}
		if p.IsVolatile() {
			fmt.Fprintf(&b, " volatile")
		}

		b.WriteString("\n")
	}

	b.WriteString("\n--- Queries ---\n")

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]
		fmt.Fprintf(&b, "  %s: %s via %s (%s)", name, q.QueryADT(),
			q.QueryEngine().Profile().Name, q.QueryComplexity())

		if s.plan != nil {
			for _, qa := range s.plan.Queries {
				if qa.QueryName == name {
					if qa.Cost.Volume > 0 {
						fmt.Fprintf(&b, " est=%.3fms", qa.Cost.EstimatedLatencyMs)
					}

					if rc := q.QueryEngine().Profile().ReadCosts; rc.NsPerPointLookup > 0 ||
						rc.NsPerFilteredScan > 0 || rc.NsPerAggregate > 0 || rc.NsPerScan > 0 {
						fmt.Fprintf(
							&b,
							" read=%.0fns",
							q.QueryEngine().Profile().NsForRead(qa.ReadPattern),
						)
					}
				}
			}
		}

		b.WriteString("\n")
	}

	if s.plan != nil && len(s.plan.Diagnostics) > 0 {
		b.WriteString("\n--- Diagnostics ---\n")

		for _, d := range s.plan.Diagnostics {
			fmt.Fprintf(&b, "  %s\n", d)
		}
	}

	return b.String()
}

// Doctor returns a runtime diagnostic report combining health checks,
// collection stats, and poisoned-collection detection. Use for debugging,
// startup diagnostics, or readiness probes.
func (s *Store) Doctor(ctx context.Context) string {
	var b strings.Builder

	b.WriteString("=== Metaengine Doctor ===\n\n--- Health ---\n")

	if err := s.HealthCheck(ctx); err != nil {
		fmt.Fprintf(&b, "  UNHEALTHY: %v\n", err)
	} else {
		b.WriteString("  all engines healthy\n")
	}

	b.WriteString("\n--- Collections ---\n")

	stats, err := s.Stats(ctx)
	if err != nil {
		fmt.Fprintf(&b, "  ERROR: %v\n", err)
	} else {
		for _, stat := range stats {
			rowStr := "N/A"
			if stat.RowCount >= 0 {
				rowStr = strconv.FormatInt(stat.RowCount, 10)
			}

			fmt.Fprintf(&b, "  %s: %s rows (%s)\n", stat.Name, rowStr, stat.EngineName)
		}
	}

	b.WriteString("\n--- Poisoned ---\n")

	poisonedAny := false

	s.poison.mu.RLock()
	for col, err := range s.poison.m {
		fmt.Fprintf(&b, "  %s: %v\n", col, err)
		poisonedAny = true
	}
	s.poison.mu.RUnlock()

	if !poisonedAny {
		b.WriteString("  none\n")
	}

	b.WriteString("\n--- Replication ---\n")

	replicatedAny := false
	for _, c := range s.Collections() {
		if c.Replication == ReplicationNone {
			continue
		}

		fmt.Fprintf(&b, "  %s: %s (lag=%dms, rtt=%dms)\n",
			c.Name, c.Replication, c.ReplicationLagMs, c.NetworkRTTMs)
		replicatedAny = true
	}

	if !replicatedAny {
		b.WriteString("  none\n")
	}

	b.WriteString("\n--- Persistence ---\n")

	volatileAny := false
	for _, c := range s.Collections() {
		if c.Persistence != PersistenceVolatile {
			continue
		}

		fmt.Fprintf(&b, "  %s: volatile (engine=%s)\n", c.Name, c.EngineName)
		volatileAny = true
	}

	if !volatileAny {
		b.WriteString("  all persistent\n")
	}

	return b.String()
}
