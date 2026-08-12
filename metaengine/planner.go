package metaengine

import (
	"fmt"
	"sort"
	"time"
)

type rankedEngine struct {
	engine     Engine
	complexity Complexity
	cost       CostEstimate
}

// DefaultWriteAmplificationBudget is the default maximum number of projections
// an event may update before the planner emits a write amplification warning.
const DefaultWriteAmplificationBudget = 3

type planConfig struct {
	writeAmplificationBudget int
	dryRun                   bool
	stats                    map[string]WorkloadStats
	replicationOverride      *Replication   // overrides all engines' declared replication for cost estimation
	networkRTTOverride       *time.Duration // overrides all engines' declared NetworkRTT for cost estimation
}

type planOption func(*planConfig)

// WithWriteAmplificationBudget sets the maximum number of projections an event
// may update without triggering a write amplification warning.
func WithWriteAmplificationBudget(n int) planOption {
	return func(c *planConfig) { c.writeAmplificationBudget = n }
}

// WithDryRun returns a planOption that skips DDL creation and engine pinning —
// Plan() returns the PlanResult (cost estimates, engine assignments, auto-
// generated layout plans) without modifying any engine state. Useful for
// inspecting what the planner would do before committing.
func WithDryRun() planOption {
	return func(c *planConfig) { c.dryRun = true }
}

// WithWorkloadStats provides observed workload statistics to the planner.
// When present, the planner emits materialize-vs-replay recommendations
// as INFO/WARN diagnostics.
//
// The map is keyed by query name. Queries without stats entries are
// skipped during materialization analysis.
func WithWorkloadStats(stats map[string]WorkloadStats) planOption {
	return func(c *planConfig) { c.stats = stats }
}

// WithReplication overrides the replication mode declared by all engines
// for cost estimation. This is a plan-time "what-if" tool: it changes the
// cost estimate (latency includes NetworkRTT) but does NOT change the
// engine's actual runtime behavior or diagnostics.
//
// Use this when the deployment topology differs from the engine's declared
// profile, or to simulate what a replicated deployment would cost.
func WithReplication(r Replication) planOption {
	return func(c *planConfig) { c.replicationOverride = &r }
}

// WithNetworkRTT overrides the network round-trip time for all engines.
// This adds a fixed per-query latency overhead to the cost estimate.
// Use this when the engine's actual network distance differs from its
// declared profile (e.g., Postgres in a different region).
func WithNetworkRTT(rtt time.Duration) planOption {
	return func(c *planConfig) { c.networkRTTOverride = &rtt }
}

// Plan creates a storage plan from available engines and declared queries.
// Each query gets its own independent projection — the same event updates
// each matching query's projection separately.
func Plan(engines []Engine, args ...any) (*Store, error) {
	if len(engines) == 0 {
		return nil, errNoEngine
	}

	var queries []any

	cfg := planConfig{writeAmplificationBudget: DefaultWriteAmplificationBudget}

	for _, arg := range args {
		switch a := arg.(type) {
		case planOption:
			a(&cfg)
		default:
			queries = append(queries, a)
		}
	}

	if len(queries) == 0 {
		return nil, errNoQuery
	}

	plan := &PlanResult{}
	store := &Store{
		engines:     engines,
		queries:     make(map[string]queryMeta),
		byInputType: make(map[string]string),
		queryDecls:  queries,
		poison:      newPoisonTracker(),
		idempotency: newIdempotencyTracker(),
		meter:       newWorkloadMeter(),
		subs:        newSubscriberHub(),
	}

	for _, q := range queries {
		meta, ok := asQueryMeta(q)
		if !ok {
			return nil, fmt.Errorf("%w: %T", errNotQueryMeta, q)
		}

		if _, exists := store.queries[meta.QueryName()]; exists {
			return nil, fmt.Errorf("%w: %q", errDuplicateQuery, meta.QueryName())
		}

		assignment, err := planQuery(meta, engines, cfg)
		if err != nil {
			return nil, fmt.Errorf("metaengine.Plan: %w", err)
		}

		store.queries[meta.QueryName()] = meta
		store.byInputType[meta.QueryInputTypeName()] = meta.QueryName()

		plan.Queries = append(plan.Queries, assignment)
	}

	pipeline := NewRulePipeline(defaultRules(cfg)...)
	if err := pipeline.Apply(plan, PlanContext{Store: store, Config: cfg}); err != nil {
		return nil, fmt.Errorf("metaengine.Plan: %w", err)
	}

	store.plan = plan
	plan.Version = 1
	plan.ComputedAt = time.Now()

	return store, nil
}

func planQuery(meta queryMeta, engines []Engine, pc planConfig) (QueryAssignment, error) {
	folds := meta.QueryFolds()
	adt := meta.QueryADT()
	cfg := meta.QueryConfig()

	if !adt.Valid() {
		return QueryAssignment{}, fmt.Errorf(
			"%w: %q has ADT %q",
			errInvalidADT,
			meta.QueryName(),
			adt,
		)
	}
	if rp := meta.QueryReadPattern(); !rp.Valid() {
		return QueryAssignment{}, fmt.Errorf(
			"%w: %q has ReadPattern %q", errInvalidReadPattern, meta.QueryName(), rp,
		)
	}
	for i, f := range folds {
		if !f.Kind().Valid() {
			return QueryAssignment{}, fmt.Errorf(
				"%w: %q fold[%d] has FoldKind %q",
				errInvalidFoldKind,
				meta.QueryName(),
				i,
				f.Kind(),
			)
		}
	}

	foldByEvent := make(map[string]int, len(folds))
	for i, f := range folds {
		if f.Kind() != FoldSkip {
			foldByEvent[f.EventType()] = i
		}
	}

	var ranked []rankedEngine

	for _, eng := range engines {
		if c, ok := eng.Profile().SupportsADT(adt); ok {
			profile := eng.Profile()

			if pc.replicationOverride != nil {
				profile.Replication = *pc.replicationOverride
			}

			if pc.networkRTTOverride != nil {
				profile.NetworkRTT = *pc.networkRTTOverride
			}

			readC := effectiveReadComplexity(meta.QueryReadPattern(), c)
			ranked = append(ranked, rankedEngine{
				engine:     eng,
				complexity: c,
				cost: estimateCost(
					readC,
					cfg.Volume,
					profile.NsForRead(meta.QueryReadPattern()),
					profile.NetworkRTT,
				),
			})
		}
	}

	assignment := QueryAssignment{
		QueryName:   meta.QueryName(),
		ADT:         adt,
		ReadPattern: meta.QueryReadPattern(),
		IsPaginated: meta.QueryIsPaginated(),
	}

	if len(ranked) == 0 {
		return QueryAssignment{}, fmt.Errorf(
			"query %q requires ADT %s but %w — add a Memory engine (supports all ADTs) or declare it as degraded on an existing engine via DegradedADTs",
			meta.QueryName(),
			adt,
			errADTNotSupported,
		)
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].cost.EstimatedLatencyMs != ranked[j].cost.EstimatedLatencyMs {
			return ranked[i].cost.EstimatedLatencyMs < ranked[j].cost.EstimatedLatencyMs
		}

		return complexityRank(ranked[i].complexity) < complexityRank(ranked[j].complexity)
	})

	best := ranked[0]
	assignment.EngineName = best.engine.Profile().Name
	assignment.Complexity = best.complexity
	assignment.Cost = best.cost

	assignment.Diagnostics = planDiagnostics(meta, best, cfg)

	meta.assignPlan(best.engine, best.complexity, foldByEvent)

	return assignment, nil
}

func planDiagnostics(meta queryMeta, best rankedEngine, cfg QueryConfig) []Diagnostic {
	var diags []Diagnostic

	if meta.QueryADT() == ADTGraph && best.complexity == ComplexityON {
		diags = append(diags, Diagnostic{
			Level:   DiagLevelDegraded,
			Query:   meta.QueryName(),
			Message: "graph traversal via scan (O(N)). Add a graph engine for O(degree^depth).",
		})
	}

	if meta.QueryIsPaginated() && best.complexity == ComplexityON {
		diags = append(diags, Diagnostic{
			Level:   DiagLevelDegraded,
			Query:   meta.QueryName(),
			Message: "filtered scan via in-memory O(N). SQLite with FilterOnField/SortOnField offers O(logN) via json_extract pushdown.",
		})
	}

	if cfg.LatencyBudgetMs > 0 && !best.cost.WithinBudget(cfg.LatencyBudgetMs) {
		diags = append(diags, Diagnostic{
			Level: DiagLevelWarn,
			Query: meta.QueryName(),
			Message: fmt.Sprintf(
				"estimated latency %.3fms exceeds budget %dms",
				best.cost.EstimatedLatencyMs, cfg.LatencyBudgetMs,
			),
		})
	}

	if diag := checkScaleThreshold(meta.QueryADT(), cfg.Volume); diag != nil {
		diag.Query = meta.QueryName()
		diags = append(diags, *diag)
	}

	return diags
}

func complexityRank(c Complexity) int {
	switch c {
	case ComplexityO1:
		return 0
	case ComplexityOLogN:
		return 1
	case ComplexityON:
		return 2
	case ComplexityONLogN:
		return 3
	case ComplexityODegree:
		return 4
	default:
		return 99
	}
}
