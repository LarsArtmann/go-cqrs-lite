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

// ModelAssignment shows which engine was assigned to a ReadModel.
type ModelAssignment struct {
	ModelName  string
	ADT        ADT
	EngineName string
	Complexity Complexity
}

func (a ModelAssignment) String() string {
	return fmt.Sprintf("%s: %s via %s (%s)", a.ModelName, a.ADT, a.EngineName, a.Complexity)
}

// QueryAssignment shows the read pattern for a specific query.
type QueryAssignment struct {
	QueryName   string
	ModelName   string
	ReadPattern ReadPattern
	Complexity  Complexity
	Filters     []FieldPath
	SortField   string
	IsPaginated bool
	Diagnostics []Diagnostic
}

func (a QueryAssignment) String() string {
	parts := []string{
		fmt.Sprintf("%s → %s: %s (%s)", a.QueryName, a.ModelName, a.ReadPattern, a.Complexity),
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
	Models      []ModelAssignment
	Queries     []QueryAssignment
	Diagnostics Diagnostics
}

func (p PlanResult) Report() string {
	var b strings.Builder
	b.WriteString("=== Meta-Engine Plan ===\n\n")

	b.WriteString("--- Models ---\n")
	for _, a := range p.Models {
		fmt.Fprintf(&b, "  %s\n", a)
	}

	if len(p.Queries) > 0 {
		b.WriteString("\n--- Queries ---\n")
		for _, a := range p.Queries {
			fmt.Fprintf(&b, "  %s\n", a)
			for _, d := range a.Diagnostics {
				fmt.Fprintf(&b, "    %s\n", d)
			}
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
// ReadModels referenced by multiple queries are deduplicated — one engine
// assignment per model, eliminating write amplification.
func Plan(engines []Engine, queries ...any) (*Store, error) {
	if len(engines) == 0 {
		return nil, errors.New("metaengine.Plan: at least one engine required")
	}

	if len(queries) == 0 {
		return nil, errors.New("metaengine.Plan: at least one query required")
	}

	plan := &PlanResult{}
	store := &Store{
		engines:     engines,
		models:      make(map[string]modelRuntime),
		queries:     make(map[string]queryRuntime),
		byInputType: make(map[string]string),
	}

	// Phase 1: Assign engines to unique models.
	modelSeen := make(map[string]bool)

	for _, q := range queries {
		meta, ok := q.(queryMeta)
		if !ok {
			return nil, fmt.Errorf(
				"query %T does not implement queryMeta — pass a metaengine.Query[Q,R]", q,
			)
		}

		rm := meta.QueryModel()
		if modelSeen[rm.Name] {
			continue
		}

		modelSeen[rm.Name] = true

		mr, assignment, err := planModel(rm, engines)
		if err != nil {
			return nil, fmt.Errorf("metaengine.Plan: %w", err)
		}

		store.models[rm.Name] = mr
		plan.Models = append(plan.Models, assignment)
	}

	// Phase 2: Register queries against their models.
	for _, q := range queries {
		meta := q.(queryMeta)
		rm := meta.QueryModel()

		modelRT, ok := store.models[rm.Name]
		if !ok {
			return nil, fmt.Errorf("metaengine.Plan: model %q not found (internal error)", rm.Name)
		}

		qr := queryRuntime{
			name:          meta.QueryName(),
			modelName:     rm.Name,
			readPattern:   meta.QueryReadPattern(),
			filters:       meta.QueryFilters(),
			sortField:     meta.QuerySortField(),
			isPaginated:   meta.QueryIsPaginated(),
			inputTypeName: meta.QueryInputTypeName(),
		}

		store.queries[qr.name] = qr
		store.byInputType[qr.inputTypeName] = qr.name

		assignment := QueryAssignment{
			QueryName:   qr.name,
			ModelName:   rm.Name,
			ReadPattern: qr.readPattern,
			Complexity:  modelRT.complexity,
			Filters:     qr.filters,
			SortField:   qr.sortField,
			IsPaginated: qr.isPaginated,
		}

		if rm.ADT == ADTGraph && modelRT.complexity == ComplexityON {
			assignment.Diagnostics = append(assignment.Diagnostics, Diagnostic{
				Level:   DiagLevelDegraded,
				Query:   qr.name,
				Message: "graph traversal via scan (O(N)). Add a graph engine for O(degree^depth).",
			})
		}

		if qr.isPaginated && modelRT.complexity == ComplexityON {
			assignment.Diagnostics = append(assignment.Diagnostics, Diagnostic{
				Level:   DiagLevelDegraded,
				Query:   qr.name,
				Message: "filtered scan via in-memory O(N). Add SQLite for O(logN) indexed scanning.",
			})
		}

		plan.Queries = append(plan.Queries, assignment)
	}

	plan.Diagnostics = checkWriteAmplification(plan.Models)
	store.plan = plan

	return store, nil
}

func planModel(rm ReadModel, engines []Engine) (modelRuntime, ModelAssignment, error) {
	folds := rm.Folds

	foldByEvent := make(map[string]int, len(folds))
	for i, f := range folds {
		if f.Kind != FoldSkip {
			foldByEvent[f.EventType] = i
		}
	}

	var ranked []rankedEngine
	for _, eng := range engines {
		if c, ok := eng.Profile().SupportsADT(rm.ADT); ok {
			ranked = append(ranked, rankedEngine{engine: eng, complexity: c})
		}
	}

	assignment := ModelAssignment{
		ModelName: rm.Name,
		ADT:       rm.ADT,
	}

	if len(ranked) == 0 {
		return modelRuntime{}, assignment, fmt.Errorf(
			"model %q requires ADT %s but no engine supports it",
			rm.Name, rm.ADT,
		)
	}

	sort.Slice(ranked, func(i, j int) bool {
		return complexityRank(ranked[i].complexity) < complexityRank(ranked[j].complexity)
	})

	best := ranked[0]
	assignment.EngineName = best.engine.Profile().Name
	assignment.Complexity = best.complexity

	mr := modelRuntime{
		name:        rm.Name,
		adt:         rm.ADT,
		engine:      best.engine,
		folds:       folds,
		foldByEvent: foldByEvent,
		complexity:  best.complexity,
	}

	return mr, assignment, nil
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

func checkWriteAmplification(models []ModelAssignment) Diagnostics {
	var diags Diagnostics
	if len(models) > 5 {
		diags = append(diags, Diagnostic{
			Level:   DiagLevelWarn,
			Query:   "*",
			Message: fmt.Sprintf("%d models — high write amplification. Consider sharing read models.", len(models)),
		})
	}

	return diags
}
