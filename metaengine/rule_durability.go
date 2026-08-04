package metaengine

import "fmt"

// durabilityRule surfaces persistence metadata as planning diagnostics.
// When a query is routed to a volatile engine (data lost on process exit),
// the rule emits:
//   - WARN if no persistent alternative exists for the same ADT.
//   - INFO if a persistent alternative exists, showing the engine name and
//     complexity so the operator can decide whether the speed gain is worth
//     the restart-rebuild cost.
//
// Persistence is orthogonal to replication (topology) and durability tiers
// (fsync). This rule makes the survivability dimension visible in EXPLAIN
// output instead of hiding it inside an opaque EngineProfile.
type durabilityRule struct{}

func (*durabilityRule) Name() string { return "durability" }

func (r *durabilityRule) Apply(result *PlanResult, ctx PlanContext) error {
	for _, q := range result.Queries {
		meta, ok := ctx.Store.queries[q.QueryName]
		if !ok {
			continue
		}

		profile := meta.QueryEngine().Profile()
		if !profile.IsVolatile() {
			continue
		}

		adt := meta.QueryADT()

		// Check if any other engine in the plan is persistent and supports the same ADT.
		var altName string
		var altComplexity Complexity
		var altNsPerOp float64
		for _, eng := range ctx.Store.engines {
			ep := eng.Profile()
			if ep.Name == profile.Name {
				continue
			}
			if !ep.IsPersistent() {
				continue
			}
			if c, supported := ep.SupportsADT(adt); supported {
				altName = ep.Name
				altComplexity = c
				altNsPerOp = ep.NsPerOp
				break
			}
		}

		if altName != "" {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Level: DiagLevelInfo,
				Query: q.QueryName,
				Message: fmt.Sprintf(
					"routed to volatile engine %q — data lost on restart (persistent alternative: %s at %s, %.0fns/op)",
					profile.Name, altName, altComplexity, altNsPerOp,
				),
			})
		} else {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Level: DiagLevelWarn,
				Query: q.QueryName,
				Message: fmt.Sprintf(
					"routed to volatile engine %q — projection will be lost on restart and must be rebuilt from the event log",
					profile.Name,
				),
			})
		}

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  q.QueryName,
			Reason: "volatile engine " + profile.Name,
		})
	}

	return nil
}
