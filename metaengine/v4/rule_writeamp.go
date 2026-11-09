package metaengine

// writeAmpRule detects write amplification: when a single event type updates
// more projections than the configured budget, the planner emits a warning.
// This helps consumers identify events that fan out to too many projections,
// which can cause write latency.
type writeAmpRule struct {
	budget int
}

func (*writeAmpRule) Name() string { return "write-amplification" }

func (r *writeAmpRule) Apply(result *PlanResult, ctx PlanContext) error {
	diags := checkWriteAmplification(ctx.Store.queries, r.budget)
	result.Diagnostics = append(result.Diagnostics, diags...)

	return nil
}
