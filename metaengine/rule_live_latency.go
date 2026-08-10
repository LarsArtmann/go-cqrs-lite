package metaengine

import "fmt"

// liveLatencyRule warns when the plan routes queries to a remote engine whose
// RTT is not backed by a fresh live measurement. This is the "graceful
// degradation, never silent lies" signal from the live-latency model
// (docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md §4.2f): a prior or stale RTT
// is still usable for routing (better than refusing to route), but the operator
// must be told the estimate is not current.
//
// The rule emits one WARN per remote engine that lacks fresh measurement,
// scoped to the queries routed to it. It does NOT change assignments — that is
// the cost-based ranker's job.
type liveLatencyRule struct{}

func (*liveLatencyRule) Name() string { return "live-latency" }

func (r *liveLatencyRule) Apply(result *PlanResult, ctx PlanContext) error {
	if ctx.Store == nil {
		return nil
	}

	// queriesByEngine maps engine name → sorted query names routed to it.
	queriesByEngine := make(map[string][]string)
	for _, q := range result.Queries {
		queriesByEngine[q.EngineName] = append(queriesByEngine[q.EngineName], q.QueryName)
	}

	ctx.Store.mu.RLock()
	engines := ctx.Store.engines
	ctx.Store.mu.RUnlock()

	for _, eng := range engines {
		profile := eng.Profile()
		if !profile.IsRemote() {
			continue
		}

		reporter, hasReporter := eng.(liveLatencyReporter)
		fresh := false
		if hasReporter {
			fresh = reporter.LiveLatency().Fresh
		}

		if fresh {
			continue
		}

		// Emit one diagnostic per engine, listing the affected queries so the
		// operator can see the blast radius. Keep the message actionable: tell
		// them to start probing.
		queries := queriesByEngine[profile.Name]
		if len(queries) == 0 {
			// Remote engine that no query uses — still surface it once so a
			// future re-plan does not silently trust the prior.
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Level: DiagLevelWarn,
				Query: "*",
				Message: fmt.Sprintf(
					"engine %q is remote (rtt=%s prior) but has no live RTT measurement — "+
						"estimates rely on a compile-time prior; call ProbeEngine to measure",
					profile.Name, profile.NetworkRTT,
				),
			})

			continue
		}

		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Level: DiagLevelWarn,
			Query: queries[0],
			Message: fmt.Sprintf(
				"routed to remote engine %q on %s RTT (%s) — estimates are not live-measured; "+
					"call ProbeEngine to replace the prior with a measurement",
				profile.Name, priorOrStale(hasReporter), profile.NetworkRTT,
			),
		})

		result.RuleTrace = append(result.RuleTrace, RuleTraceEntry{
			Rule:   r.Name(),
			Query:  queries[0],
			Reason: fmt.Sprintf("%s remote, no fresh live RTT", profile.Name),
		})
	}

	return nil
}

// priorOrStale labels whether the fallback value is an untouched prior or a
// measurement that has since gone stale.
func priorOrStale(hasReporter bool) string {
	if hasReporter {
		return "stale"
	}

	return "prior"
}
