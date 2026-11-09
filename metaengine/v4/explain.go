package metaengine

import (
	"context"
	"fmt"
	"maps"
	"slices"
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

// ExplainAggregate returns the SQL that would execute for an aggregate query,
// without running it. The engine must implement ExplainableAggregate; otherwise
// a fallback message is returned.
//
//	reader := metaengine.NewReader[V](store, "find_user")
//	sql, args := reader.ExplainAggregate(ctx, metaengine.ExplainAggregateOptions{
//	    Fn: metaengine.AggregateSum, Column: "price",
//	    GroupBy: "status",
//	    Filters: []metaengine.FilterSpec{{Column: "status", Op: metaengine.FilterEq, Value: "open"}},
//	})
//	fmt.Println(sql, args)
func (r *TypedReader[V]) ExplainAggregate(
	ctx context.Context,
	opts ExplainAggregateOptions,
) (string, []any) {
	eng, ok := r.store.collectionEngine(r.collection)
	if !ok {
		return "-- no engine for collection " + r.collection, nil
	}

	if ea, ok := eng.(ExplainableAggregate); ok {
		return ea.ExplainAggregateQuery(ctx, r.collection, opts)
	}

	return "-- EXPLAIN AGGREGATE not supported by engine " + eng.Profile().Name, nil
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
	//art-dupl:accept read-locked engine iteration prologue — idiomatic store snapshot
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
			fmt.Fprintf(&b, " replication=%s, lag=%s",
				p.Replication, p.EffectiveReplicationLag())
		}
		if p.IsRemote() {
			fmt.Fprintf(&b, " %s", FormatLiveLatency(buildEngineStats(eng)))
		}
		if p.IsVolatile() {
			fmt.Fprintf(&b, " volatile")
		}

		b.WriteString("\n")
	}

	b.WriteString(explainCapabilityWarnings(s.engines))
	b.WriteString(explainVectorSection(s.engines))

	b.WriteString("\n--- Queries ---\n")

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]
		profile := q.QueryEngine().Profile()
		fmt.Fprintf(&b, "  %s: %s via %s (%s)%s", name, q.QueryADT(),
			profile.Name, q.QueryComplexity(),
			layoutExplainAnnotation(s.priorityConfig, profile, name, q.QueryConfig()))

		if s.plan != nil {
			for _, qa := range s.plan.Queries {
				if qa.QueryName == name {
					if qa.Cost.Volume > 0 {
						fmt.Fprintf(&b, " est=%.3fms", qa.Cost.EstimatedLatencyMs)
					}

					if rc := profile.ReadCosts; rc.NsPerPointLookup > 0 ||
						rc.NsPerFilteredScan > 0 || rc.NsPerAggregate > 0 || rc.NsPerScan > 0 {
						fmt.Fprintf(&b, " read=%.0fns", profile.NsForRead(qa.ReadPattern))
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

// QueryPlacement is the machine-readable answer to "where did the planner
// put this query, and what hint informed it": the assigned engine, the ADT,
// the DECLARED volume hint ([Volume], events/sec — 0 when none), and the
// plan's estimated read latency (0 while unplanned; the planner may assume
// a default volume when none is declared). Composition layers
// (system.Explain, Doctor dashboards) render this so the cost-based
// placement is introspectable without parsing ExplainPlan's text output.
type QueryPlacement struct {
	QueryName          string
	Engine             string
	ADT                string
	Volume             int64
	EstimatedLatencyMs float64
	Complexity         Complexity
}

// QueryPlacements returns one [QueryPlacement] per planned query, sorted by
// query name (the same order ExplainPlan renders). Queries not yet planned
// (no engine assigned) are returned with an empty Engine so callers can see
// the declared hints regardless of plan state.
func (s *Store) QueryPlacements() []QueryPlacement {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]QueryPlacement, 0, len(s.queries))

	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		q := s.queries[name]

		placement := QueryPlacement{
			QueryName:  name,
			ADT:        string(q.QueryADT()),
			Volume:     q.QueryConfig().Volume,
			Complexity: q.QueryComplexity(),
		}

		if eng := q.QueryEngine(); eng != nil {
			placement.Engine = eng.Profile().Name
		}

		if s.plan != nil {
			for _, qa := range s.plan.Queries {
				if qa.QueryName == name && qa.Cost.Volume > 0 {
					placement.EstimatedLatencyMs = qa.Cost.EstimatedLatencyMs

					break
				}
			}
		}

		out = append(out, placement)
	}

	return out
}

// Doctor returns a runtime diagnostic report combining health checks,
// collection stats, and poisoned-collection detection. Use for debugging,
// startup diagnostics, or readiness probes.
func (s *Store) Doctor(ctx context.Context) string {
	var b strings.Builder

	b.WriteString("=== Metaengine Doctor ===\n\n--- Health ---\n")
	s.writeDoctorHealthSection(ctx, &b)

	b.WriteString("\n--- Engine Health (ADR-0137) ---\n" + s.doctorEngineHealthSection())
	s.writeDoctorCatchUpSection(&b)

	b.WriteString("\n--- Collections ---\n")
	s.writeDoctorCollectionsSection(ctx, &b)

	b.WriteString("\n--- Poisoned ---\n")
	s.writeDoctorPoisonedSection(&b)

	b.WriteString("\n--- Durability ---\n")
	engines := s.writeDoctorDurabilitySection(&b)

	b.WriteString(vectorDoctorSection(ctx, engines))

	b.WriteString("\n--- Replication ---\n")
	s.writeDoctorReplicationSection(&b)

	b.WriteString("\n--- Persistence ---\n")
	s.writeDoctorPersistenceSection(&b)

	b.WriteString("\n--- Aggregate Pushdown ---\n")
	s.writeDoctorAggregateSection(&b)

	b.WriteString("\n--- Latency ---\n")
	s.writeDoctorLatencySection(ctx, &b)

	b.WriteString(s.degradedDoctorSection())

	b.WriteString("\n--- Routing ---\n")
	s.writeDoctorRoutingSection(ctx, &b)

	b.WriteString(s.capabilityDoctorSection())
	b.WriteString(s.recordContextDoctorSection())
	b.WriteString(s.LayoutDoctorSection())
	b.WriteString(s.PlannedTablesDoctorSection(ctx))
	b.WriteString(s.MaterializedViewsDoctorSection(ctx))
	b.WriteString(s.resetDoctorSection())

	return b.String()
}

// aggregateCapabilities returns a human-readable description of the aggregate
// pushdown interfaces an engine implements. Returns "" if the engine has no
// aggregate pushdown support.
func aggregateCapabilities(eng Engine) string {
	var caps []string

	if _, ok := eng.(AggregateReader); ok {
		caps = append(caps, "scalar")
	}

	if _, ok := eng.(GroupedAggregateReader); ok {
		caps = append(caps, "grouped")
	}

	if _, ok := eng.(MultiAggregateReader); ok {
		caps = append(caps, "multi")
	}

	if _, ok := eng.(MultiGroupedAggregateReader); ok {
		caps = append(caps, "multi-grouped")
	}

	if _, ok := eng.(DistinctReader); ok {
		caps = append(caps, "distinct")
	}

	if len(caps) == 0 {
		return ""
	}

	return "pushdown: " + strings.Join(caps, ", ")
}
