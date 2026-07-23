package metaengine

import (
	"fmt"
	"sort"
	"strings"
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
		if diag.Level == "WARN" || diag.Level == "DEGRADED" {
			return true
		}
	}
	return false
}

type QueryAssignment struct {
	QueryName   string
	ADT         ADT
	ReadPattern ReadPattern
	EngineName  string
	Complexity  Complexity
	Filters     []FieldPath
	SortField   string
	IsPaginated bool
	Diagnostics []Diagnostic
}

func (a QueryAssignment) String() string {
	parts := []string{
		fmt.Sprintf("%s: %s via %s (%s)", a.QueryName, a.ADT, a.EngineName, a.Complexity),
	}
	if len(a.Filters) > 0 {
		names := make([]string, len(a.Filters))
		for i, f := range a.Filters {
			names[i] = f.Field
		}
		parts = append(parts, "filter=["+strings.Join(names, ",")+"]")
	}
	if a.SortField != "" {
		parts = append(parts, "sort="+a.SortField)
	}
	if a.IsPaginated {
		parts = append(parts, "[paginated]")
	}
	return strings.Join(parts, " ")
}

type PlanResult struct {
	Assignments []QueryAssignment
	Diagnostics Diagnostics
}

func (p PlanResult) Report() string {
	var b strings.Builder
	b.WriteString("=== Meta-Engine Plan ===\n\n")
	for _, a := range p.Assignments {
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
}

// Plan creates a storage plan from available engines and declared queries.
func Plan(engines []Engine, queries ...any) (*Store, error) {
	if len(engines) == 0 {
		return nil, fmt.Errorf("metaengine.Plan: at least one engine required")
	}
	if len(queries) == 0 {
		return nil, fmt.Errorf("metaengine.Plan: at least one query required")
	}

	plan := &PlanResult{}
	store := &Store{
		engines:     engines,
		queries:     make(map[string]queryRuntime),
		byInputType: make(map[string]string),
	}

	for _, q := range queries {
		runtime, assignment, err := planQuery(q, engines)
		if err != nil {
			return nil, fmt.Errorf("metaengine.Plan: %w", err)
		}
		store.queries[runtime.name] = runtime
		store.byInputType[runtime.inputTypeName] = runtime.name
		plan.Assignments = append(plan.Assignments, assignment)
	}

	plan.Diagnostics = checkWriteAmplification(plan.Assignments)
	store.plan = plan
	return store, nil
}

func planQuery(q any, engines []Engine) (queryRuntime, QueryAssignment, error) {
	meta, ok := q.(queryMeta)
	if !ok {
		return queryRuntime{}, QueryAssignment{}, fmt.Errorf(
			"query %T does not implement queryMeta — pass a metaengine.Query[Q,R]", q,
		)
	}

	folds := meta.QueryFolds()
	foldByEvent := make(map[string]int, len(folds))
	for i, f := range folds {
		if f.Kind != FoldSkip {
			foldByEvent[f.EventType] = i
		}
	}

	runtime := queryRuntime{
		name:          meta.QueryName(),
		adt:           meta.QueryADT(),
		readPattern:   meta.QueryReadPattern(),
		filters:       meta.QueryFilters(),
		sortField:     meta.QuerySortField(),
		isPaginated:   meta.QueryIsPaginated(),
		folds:         folds,
		foldByEvent:   foldByEvent,
		inputTypeName: meta.QueryInputTypeName(),
	}

	var ranked []rankedEngine
	for _, eng := range engines {
		if c, ok := eng.Profile().SupportsADT(runtime.adt); ok {
			ranked = append(ranked, rankedEngine{engine: eng, complexity: c})
		}
	}

	assignment := QueryAssignment{
		QueryName:   runtime.name,
		ADT:         runtime.adt,
		ReadPattern: runtime.readPattern,
		Filters:     runtime.filters,
		SortField:   runtime.sortField,
		IsPaginated: runtime.isPaginated,
	}

	if len(ranked) == 0 {
		return queryRuntime{}, assignment, fmt.Errorf(
			"query %q requires ADT %s but no engine supports it",
			runtime.name, runtime.adt,
		)
	}

	sort.Slice(ranked, func(i, j int) bool {
		return complexityRank(ranked[i].complexity) < complexityRank(ranked[j].complexity)
	})

	best := ranked[0]
	assignment.EngineName = best.engine.Profile().Name
	assignment.Complexity = best.complexity
	runtime.engine = best.engine

	if runtime.adt == ADTGraph && best.complexity == ComplexityON {
		assignment.Diagnostics = append(assignment.Diagnostics, Diagnostic{
			Level:   "DEGRADED",
			Query:   runtime.name,
			Message: "graph traversal via SQL/in-memory scan (O(N)). Add a graph engine for O(degree^depth).",
		})
	}
	if runtime.adt == ADTSortedMap && best.complexity == ComplexityON {
		assignment.Diagnostics = append(assignment.Diagnostics, Diagnostic{
			Level:   "DEGRADED",
			Query:   runtime.name,
			Message: "filtered scan via in-memory O(N) filter. Add SQLite for O(logN) indexed scanning.",
		})
	}

	return runtime, assignment, nil
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

func checkWriteAmplification(assignments []QueryAssignment) Diagnostics {
	var diags Diagnostics
	if len(assignments) > 5 {
		diags = append(diags, Diagnostic{
			Level: "WARN",
			Query: "*",
			Message: fmt.Sprintf(
				"%d projections — high write amplification. Consider sharing projections.",
				len(assignments),
			),
		})
	}
	return diags
}
