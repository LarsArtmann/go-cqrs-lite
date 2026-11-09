package metaengine

import "time"

type planOption func(*planConfig)

// WithWriteAmplificationBudget sets the maximum number of projections an event
// may update without triggering a write amplification warning.
func WithWriteAmplificationBudget(n int) planOption {
	return func(c *planConfig) { c.writeAmplificationBudget = n }
}

// WithEngineCapabilityGaps documents known ADT conformance gaps per engine
// (same semantics as CapabilityAudit's gaps argument). An over-declaring
// engine whose gap is documented for the query's ADT is still EXCLUDED from
// routing — the backend does not exist and execution would hard-error — but
// the plan stays silent about it instead of re-announcing the known gap on
// every plan/replan. Gaps persist across Replan.
func WithEngineCapabilityGaps(engineGaps map[string]CapabilityGaps) planOption {
	return func(c *planConfig) { c.capabilityGaps = engineGaps }
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

// WithDefaultLimit sets the OPERATOR ceiling for scans: when a caller issues
// a Scan without an explicit [WithLimit], the scan is bounded to n rows
// instead of the built-in 100. It exists so operators can re-pin the scan
// bound deployment-wide (v5 decision G-T14: the default flips to unbounded;
// this option is how an operator opts back into a bound). Values <= 0 are
// ignored — the built-in 100 stays in force. An explicit WithLimit on the
// scan always wins.
func WithDefaultLimit(n int) planOption {
	return func(c *planConfig) {
		if n > 0 {
			c.defaultLimit = n
		}
	}
}
