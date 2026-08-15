package metaengine

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// sharedCollectionRule enforces ADR-0124 §Aggregate Boundaries opt-in: child
// Go types declared shared (WithSharedCollection) must not be embedded —
// embedding duplicates the shared child into every carrying collection. The
// rule forces LayoutNormalize on affected queries and emits diagnostics; a
// shared type spanning multiple collections is surfaced as a WARN so operators
// see the cross-collection data flow they opted into (or misconfigured).
//
// v1 scope is scoring-level: physical child-collection materialization does
// not exist yet, so the rule's observable effect is the layout decision and
// diagnostics (METAENGINE-LAYOUT-ROLES.md §6).
type sharedCollectionRule struct {
	shared map[string]bool
}

func (*sharedCollectionRule) Name() string { return "shared-collection" }

func (r *sharedCollectionRule) Apply(result *PlanResult, ctx PlanContext) error {
	spanning := make(map[string][]string)

	for i := range result.Queries {
		qa := &result.Queries[i]

		q, ok := ctx.Store.queries[qa.QueryName]
		if !ok {
			continue
		}

		matches := sharedTypesInResult(q.QueryResultType(), r.shared)
		if len(matches) == 0 {
			continue
		}

		qa.Layout = LayoutNormalize
		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  qa.QueryName,
			Reason: "shared child type(s): " + strings.Join(matches, ", "),
		})

		for _, typeName := range matches {
			spanning[typeName] = append(spanning[typeName], qa.QueryName)

			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Level: DiagLevelInfo,
				Query: qa.QueryName,
				Message: fmt.Sprintf(
					"child type %q declared shared — layout forced to Normalize (embedding would duplicate the shared child)",
					typeName,
				),
			})
		}
	}

	r.warnSpanning(result, spanning)

	return nil
}

func (r *sharedCollectionRule) warnSpanning(result *PlanResult, spanning map[string][]string) {
	names := make([]string, 0, len(spanning))

	for typeName := range spanning {
		names = append(names, typeName)
	}

	sort.Strings(names)

	for _, typeName := range names {
		queries := spanning[typeName]
		if len(queries) < 2 {
			continue
		}

		sort.Strings(queries)

		msg := fmt.Sprintf(
			"shared type %q spans %d collections (%s) — without a shared collection these copies drift independently",
			typeName,
			len(queries),
			strings.Join(queries, ", "),
		)

		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Level:   DiagLevelWarn,
			Message: msg,
		})
	}
}

// sharedTypesInResult returns the distinct declared-shared type names carried
// by a query result type — directly, as *T, as []T, or as a map value
// (top-level fields only).
func sharedTypesInResult(rt reflect.Type, shared map[string]bool) []string {
	if len(shared) == 0 || rt == nil {
		return nil
	}

	rt = derefStructType(rt)
	if rt == nil {
		return nil
	}

	var out []string
	seen := make(map[string]bool)

	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		core := derefStructType(field.Type)
		if core == nil && field.Type.Kind() == reflect.Map {
			core = derefStructType(field.Type.Elem()) // map value type
		}

		if core == nil {
			continue
		}

		if name := core.Name(); name != "" && shared[name] && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}

	return out
}

// derefStructType returns the struct type behind T, *T, []T, or []*T — nil
// when the type does not bottom out at a struct.
func derefStructType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}

	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return derefStructType(t.Elem())
	case reflect.Struct:
		return t
	default:
		return nil
	}
}
