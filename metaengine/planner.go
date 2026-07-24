package metaengine

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Diagnostic levels for plan output.
const (
	DiagLevelWarn     = "WARN"
	DiagLevelDegraded = "DEGRADED"
	DiagLevelInfo     = "INFO"
)

type Diagnostic struct {
	Level   string
	Query   string
	Message string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("[%s] %s: %s", d.Level, d.Query, d.Message)
}

type Diagnostics []Diagnostic

func (d Diagnostics) HasWarnings() bool {
	for _, diag := range d {
		if diag.Level == DiagLevelWarn || diag.Level == DiagLevelDegraded {
			return true
		}
	}

	return false
}

// QueryAssignment shows the full plan for one query: engine, ADT, read pattern, cost.
type QueryAssignment struct {
	QueryName   string
	ADT         ADT
	EngineName  string
	Complexity  Complexity
	ReadPattern ReadPattern
	IsPaginated bool
	Cost        CostEstimate
	Diagnostics []Diagnostic
}

func (a QueryAssignment) String() string {
	parts := []string{
		fmt.Sprintf("%s: %s/%s via %s (%s)",
			a.QueryName, a.ADT, a.ReadPattern, a.EngineName, a.Complexity),
	}

	if a.Cost.Volume > 0 {
		parts = append(parts, fmt.Sprintf("latency<%.3fms", a.Cost.EstimatedLatencyMs))
	}

	if a.IsPaginated {
		parts = append(parts, "[paginated]")
	}

	return strings.Join(parts, " ")
}

type PlanResult struct {
	Queries     []QueryAssignment
	Diagnostics Diagnostics
}

func (p PlanResult) Report() string {
	var b strings.Builder
	b.WriteString("=== Meta-Engine Plan ===\n\n")

	b.WriteString("--- Queries ---\n")

	for _, a := range p.Queries {
		fmt.Fprintf(&b, "  %s\n", a)

		for _, d := range a.Diagnostics {
			fmt.Fprintf(&b, "    %s\n", d)
		}
	}

	if len(p.Diagnostics) > 0 {
		b.WriteString("\n--- Global Diagnostics ---\n")

		for _, d := range p.Diagnostics {
			fmt.Fprintf(&b, "  %s\n", d)
		}
	}

	return b.String()
}

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
}

type planOption func(*planConfig)

// WithWriteAmplificationBudget sets the maximum number of projections an event
// may update without triggering a write amplification warning. Pass this as a
// variadic argument to Plan:
//
//	store, _ := metaengine.Plan(engines, q1, q2,
//	    metaengine.WithWriteAmplificationBudget(5))
func WithWriteAmplificationBudget(n int) planOption {
	return func(c *planConfig) { c.writeAmplificationBudget = n }
}

// Plan creates a storage plan from available engines and declared queries.
// Each query gets its own independent projection — the same event updates
// each matching query's projection separately. Write amplification across
// many queries is reported as a diagnostic warning, not prevented by dedup.
//
// Plan accepts queries and plan options as variadic arguments, separated by type:
//
//	store, _ := metaengine.Plan(engines, q1, q2, metaengine.WithWriteAmplificationBudget(5))
func Plan(engines []Engine, args ...any) (*Store, error) {
	if len(engines) == 0 {
		return nil, errors.New("metaengine.Plan: at least one engine required")
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
		return nil, errors.New("metaengine.Plan: at least one query required")
	}

	plan := &PlanResult{}
	store := &Store{
		engines:     engines,
		queries:     make(map[string]queryRuntime),
		byInputType: make(map[string]string),
	}

	for _, q := range queries {
		meta, ok := q.(queryMeta)
		if !ok {
			return nil, fmt.Errorf(
				"query %T does not implement queryMeta — pass a metaengine.Query[Q,R]", q,
			)
		}

		if _, exists := store.queries[meta.QueryName()]; exists {
			return nil, fmt.Errorf("metaengine.Plan: duplicate query name %q", meta.QueryName())
		}

		qr, assignment, err := planQuery(meta, engines)
		if err != nil {
			return nil, fmt.Errorf("metaengine.Plan: %w", err)
		}

		store.queries[qr.name] = qr
		store.byInputType[qr.inputTypeName] = qr.name

		plan.Queries = append(plan.Queries, assignment)
	}

	plan.Diagnostics = checkWriteAmplification(store.queries, cfg.writeAmplificationBudget)
	store.plan = plan

	return store, nil
}

func planQuery(meta queryMeta, engines []Engine) (queryRuntime, QueryAssignment, error) {
	folds := meta.QueryFolds()
	adt := meta.QueryADT()
	cfg := meta.QueryConfig()

	foldByEvent := make(map[string]int, len(folds))
	for i, f := range folds {
		if f.Kind != FoldSkip {
			foldByEvent[f.EventType] = i
		}
	}

	var ranked []rankedEngine

	for _, eng := range engines {
		if c, ok := eng.Profile().SupportsADT(adt); ok {
			readC := effectiveReadComplexity(meta.QueryReadPattern(), c)
			ranked = append(ranked, rankedEngine{
				engine:     eng,
				complexity: c,
				cost:       estimateCost(readC, cfg.Volume),
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
		return queryRuntime{}, assignment, fmt.Errorf(
			"query %q requires ADT %s but no engine supports it",
			meta.QueryName(), adt,
		)
	}

	// Rank by cost (estimated latency). Ties break on complexity rank for determinism.
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

	if adt == ADTGraph && best.complexity == ComplexityON {
		assignment.Diagnostics = append(assignment.Diagnostics, Diagnostic{
			Level:   DiagLevelDegraded,
			Query:   meta.QueryName(),
			Message: "graph traversal via scan (O(N)). Add a graph engine for O(degree^depth).",
		})
	}

	if meta.QueryIsPaginated() && best.complexity == ComplexityON {
		assignment.Diagnostics = append(assignment.Diagnostics, Diagnostic{
			Level:   DiagLevelDegraded,
			Query:   meta.QueryName(),
			Message: "filtered scan via in-memory O(N). Add SQLite for O(logN) indexed scanning.",
		})
	}

	if cfg.LatencyBudgetMs > 0 && !best.cost.WithinBudget(cfg.LatencyBudgetMs) {
		assignment.Diagnostics = append(assignment.Diagnostics, Diagnostic{
			Level: DiagLevelWarn,
			Query: meta.QueryName(),
			Message: fmt.Sprintf(
				"estimated latency %.3fms exceeds budget %dms",
				best.cost.EstimatedLatencyMs, cfg.LatencyBudgetMs,
			),
		})
	}

	if diag := checkScaleThreshold(adt, cfg.Volume); diag != nil {
		diag.Query = meta.QueryName()
		assignment.Diagnostics = append(assignment.Diagnostics, *diag)
	}

	qr := queryRuntime{
		name:          meta.QueryName(),
		adt:           adt,
		engine:        best.engine,
		complexity:    best.complexity,
		folds:         folds,
		foldByEvent:   foldByEvent,
		readPattern:   meta.QueryReadPattern(),
		config:        meta.QueryConfig(),
		keyType:       meta.QueryKeyType(),
		inputTypeName: meta.QueryInputTypeName(),
	}

	return qr, assignment, nil
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

func checkWriteAmplification(queries map[string]queryRuntime, budget int) Diagnostics {
	eventCount := make(map[string]int)

	for _, q := range queries {
		seen := make(map[string]bool)
		for eventType := range q.foldByEvent {
			if !seen[eventType] {
				seen[eventType] = true
				eventCount[eventType]++
			}
		}
	}

	var maxAmp int
	for _, count := range eventCount {
		if count > maxAmp {
			maxAmp = count
		}
	}

	var diags Diagnostics

	if maxAmp > budget {
		var heavy []string

		for evt, count := range eventCount {
			if count == maxAmp {
				heavy = append(heavy, evt)
			}
		}

		sort.Strings(heavy)

		diags = append(diags, Diagnostic{
			Level: DiagLevelWarn,
			Query: "*",
			Message: fmt.Sprintf(
				"events %s update %d projections — high write amplification",
				strings.Join(heavy, ", "), maxAmp,
			),
		})
	}

	return diags
}
