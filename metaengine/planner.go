package metaengine

import (
	"fmt"
	"sort"
	"time"
)

type rankedEngine struct {
	engine            Engine
	complexity        Complexity
	readComplexity    Complexity // effective READ complexity (scan degradation applied) — the axis the cost model used
	cost              CostEstimate
	weightedLatencyMs float64 // priority-adjusted latency for ranking (ADR-0124)
}

// DefaultWriteAmplificationBudget is the default maximum number of projections
// an event may update before the planner emits a write amplification warning.
const DefaultWriteAmplificationBudget = 3

// DefaultIdempotencyCapacity is the default dedup window for
// Store.ApplyIdempotent: 131072 event IDs (~10 MB worst case). Large enough
// that realistic duplicate-delivery windows (retries, at-least-once redrive)
// are covered, small enough that a long-lived store cannot leak memory
// without bound. See WithIdempotencyCapacity.
const DefaultIdempotencyCapacity = 1 << 17

type planConfig struct {
	writeAmplificationBudget int
	dryRun                   bool
	stats                    map[string]WorkloadStats
	replicationOverride      *Replication      // overrides all engines' declared replication for cost estimation
	networkRTTOverride       *time.Duration    // overrides all engines' declared NetworkRTT for cost estimation
	routingHysteresis        float64           // min fractional improvement for re-routing suggestions
	routingMinDeltaMs        float64           // min absolute improvement (ms) for re-routing suggestions
	incumbents               map[string]string // query → currently-assigned engine (Replan only); enables hysteresis-gated re-assignment
	priority                 *PriorityConfig   // operator-driven layout priorities (ADR-0124)
	sharedCollections        map[string]bool   // child Go types shared across collections (ADR-0124 aggregate boundaries)
	idempotencyCapacity      int               // dedup ring capacity for ApplyIdempotent; <=0 → unbounded legacy mode
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
// This is a PRIOR for the initial plan, not a constant: it seeds planning
// before any live probe runs and is replaced by a live measurement (via
// ProbeEngine) once fresh samples exist. Use it when the deployment topology
// differs from the engine's declared profile (e.g., Postgres in another
// region), or to simulate a what-if RTT without spinning up a probe.
func WithNetworkRTT(rtt time.Duration) planOption {
	return func(c *planConfig) { c.networkRTTOverride = &rtt }
}

// WithRoutingHysteresis sets the minimum fractional cost improvement required
// before CheckRouting suggests re-routing a query to a different engine. For
// example, 0.15 means an alternative engine must be at least 15% cheaper.
// Defaults to DefaultRoutingHysteresis (0.20 = 20%). Lower values make the
// planner more sensitive to latency shifts but risk oscillation from jitter.
func WithRoutingHysteresis(fraction float64) planOption {
	return func(c *planConfig) {
		if fraction > 0 {
			c.routingHysteresis = fraction
		}
	}
}

// WithRoutingMinDelta sets the minimum absolute cost improvement (in
// milliseconds) required before CheckRouting suggests re-routing. This floor
// prevents re-routing on tiny absolute differences for very cheap queries
// (e.g. 0.01ms), where a 20% fractional improvement is negligible. Defaults
// to DefaultRoutingMinDelta (0.5ms).
func WithRoutingMinDelta(delta time.Duration) planOption {
	return func(c *planConfig) {
		if delta > 0 {
			c.routingMinDeltaMs = float64(delta.Microseconds()) / 1e3
		}
	}
}

// WithIdempotencyCapacity bounds the in-memory dedup window used by
// Store.ApplyIdempotent. When the window is full the oldest event IDs are
// evicted — dedup stays best-effort (duplicates older than the window
// re-apply, which is within the at-least-once contract; use an external
// idempotency store for durable dedup). Defaults to
// DefaultIdempotencyCapacity. Pass <= 0 for the legacy unbounded behavior,
// accepting the unbounded memory growth that comes with it.
func WithIdempotencyCapacity(n int) planOption {
	return func(c *planConfig) { c.idempotencyCapacity = n }
}

// Plan creates a storage plan from available engines and declared queries.
// Each query gets its own independent projection — the same event updates
// each matching query's projection separately.
func Plan(engines []Engine, args ...any) (*Store, error) {
	if len(engines) == 0 {
		return nil, errNoEngine
	}

	var queries []any

	cfg := planConfig{
		writeAmplificationBudget: DefaultWriteAmplificationBudget,
		idempotencyCapacity:      DefaultIdempotencyCapacity,
	}

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
		engines:           engines,
		queries:           make(map[string]queryMeta),
		byInputType:       make(map[string]string),
		queryDecls:        queries,
		poison:            newPoisonTracker(),
		idempotency:       newIdempotencyTracker(cfg.idempotencyCapacity),
		meter:             newWorkloadMeter(),
		subs:              newSubscriberHub(),
		foldLocks:         newFoldLocks(),
		engineRoles:       make(map[string]ProjectionRole),
		replicas:          make(map[string]*replicator),
		sharedCollections: cfg.sharedCollections,
		routingHysteresis: defaultRoutingHysteresis(cfg.routingHysteresis),
		routingMinDelta:   defaultRoutingMinDelta(cfg.routingMinDeltaMs),
	}

	for _, q := range queries {
		meta, ok := asQueryMeta(q)
		if !ok {
			return nil, fmt.Errorf("%w: %T", errNotQueryMeta, q)
		}

		if err := meta.ensureFolds(); err != nil {
			return nil, fmt.Errorf("metaengine.Plan: %w", err)
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

	store.rebuildTaskSnapLocked()

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
			cost := estimateCost(
				readC,
				cfg.Volume,
				profile.NsForRead(meta.QueryReadPattern()),
				profile.NetworkRTT,
			)

			weightedMs := cost.EstimatedLatencyMs
			if pc.priority != nil {
				p := pc.priority.Resolve(profile.Name, meta.QueryName())
				weightedMs = cost.EstimatedLatencyMs * priorityFactor(p, readC)
			}

			ranked = append(ranked, rankedEngine{
				engine:            eng,
				complexity:        c,
				readComplexity:    readC,
				cost:              cost,
				weightedLatencyMs: weightedMs,
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
		if ranked[i].weightedLatencyMs != ranked[j].weightedLatencyMs {
			return ranked[i].weightedLatencyMs < ranked[j].weightedLatencyMs
		}

		return complexityRank(ranked[i].complexity) < complexityRank(ranked[j].complexity)
	})

	best := ranked[0]

	// Hysteresis-gated re-assignment: when re-planning (Replan), keep the
	// incumbent engine unless the cheaper alternative beats it beyond BOTH
	// deadbands — the same rule checkQueryRouting applies to suggestions.
	// Pure argmin assignment flips near-parity engines back and forth on
	// every auto-replan tick (A→B→A oscillation) whenever live latency
	// jitters around the tie point.
	if incumbentName, ok := pc.incumbents[meta.QueryName()]; ok &&
		incumbentName != best.engine.Profile().Name {
		for i := range ranked {
			if ranked[i].engine.Profile().Name != incumbentName {
				continue
			}

			incumbent := ranked[i]

			// A strictly better complexity class is a structural win, not
			// measurement jitter — hysteresis never keeps the asymptotically
			// worse engine.
			if complexityRank(best.complexity) < complexityRank(incumbent.complexity) {
				break
			}

			improvement := incumbent.weightedLatencyMs - best.weightedLatencyMs

			fraction := 0.0
			if incumbent.weightedLatencyMs > 0 {
				fraction = improvement / incumbent.weightedLatencyMs
			}

			if improvement <= 0 ||
				fraction <= pc.routingHysteresis ||
				improvement < pc.routingMinDeltaMs {
				best = incumbent
			}

			break
		}
	}

	assignment.EngineName = best.engine.Profile().Name
	assignment.Complexity = best.complexity
	assignment.Cost = best.cost

	// Compute the layout decision (ADR-0124) so the plan carries it. This
	// converges engine routing and layout scoring into one planning pass —
	// ReplanLayout reads this field instead of assuming LayoutEmbed.
	resolvedPriority := resolvePriority(pc.priority, best.engine.Profile().Name, meta.QueryName(),
		cfg.layoutPriorityOr(PriorityBalanced))
	assignment.Layout, _ = SelectLayout(best.engine.Profile(), resolvedPriority)

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

	if cfg.Volume <= 0 {
		diags = append(diags, Diagnostic{
			Level: DiagLevelInfo,
			Query: meta.QueryName(),
			Message: "volume not set; cost model assumed default of 1000 items. " +
				"Set QueryConfig.Volume for accurate latency estimates.",
		})
	}

	if fc := cfg.FilterCount(); fc > 0 &&
		(best.readComplexity == ComplexityON || best.readComplexity == ComplexityONLogN) {
		sel := filterSelectivity(fc)
		diags = append(diags, Diagnostic{
			Level: DiagLevelInfo,
			Query: meta.QueryName(),
			Message: fmt.Sprintf(
				"%d filter(s) on %s scan; estimated selectivity %.4f (not applied to routing cost). "+
					"Engines with index pushdown (FilterOnField) avoid the full scan.",
				fc,
				best.readComplexity,
				sel,
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
