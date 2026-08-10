package metaengine

import (
	"context"
	"fmt"
	"time"
)

// DefaultRoutingHysteresis is the minimum fractional cost improvement required
// before CheckRouting suggests re-routing a query to a different engine. 20%
// means an alternative engine must be at least 20% cheaper than the current
// assignment before a REPLAN-SUGGESTED diagnostic is emitted. This deadband
// prevents oscillation when live RTT measurements jitter around the tie point.
const DefaultRoutingHysteresis = 0.20

// CheckRouting evaluates whether the current plan's engine assignments are
// still optimal given live latency measurements. For each query where a
// different engine is now significantly cheaper (exceeding the hysteresis
// deadband), it emits a REPLAN-SUGGESTED diagnostic advising the caller to
// invoke Store.Replan.
//
// This is the execution-time re-scoring mechanism: it does NOT change engine
// assignments or trigger I/O — it re-scores engines using their current
// Profile() (which reflects live tracker EWMA) and compares against the plan's
// recorded assignment. The caller decides whether to act on the suggestion by
// calling Replan.
//
// Use this between periodic replans, or after a latency shift is detected via
// GetEngineStats, to decide whether a full re-plan is worthwhile:
//
//	diags := store.CheckRouting(ctx)
//	if len(diags) > 0 {
//	    _ = store.Replan(ctx)
//	}
func (s *Store) CheckRouting(ctx context.Context) []Diagnostic {
	if err := ctx.Err(); err != nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.plan == nil {
		return nil
	}

	var diags []Diagnostic

	for _, qa := range s.plan.Queries {
		q, ok := s.queries[qa.QueryName]
		if !ok {
			continue
		}

		diag := checkQueryRouting(q, qa, s.engines)
		if diag != nil {
			diags = append(diags, *diag)
		}
	}

	return diags
}

// checkQueryRouting re-scores all eligible engines for a single query and
// returns a REPLAN-SUGGESTED diagnostic if a cheaper alternative exists beyond
// the hysteresis deadband. Returns nil when the current assignment is still
// optimal (or near-optimal within the deadband).
func checkQueryRouting(
	q queryMeta,
	qa QueryAssignment,
	engines []Engine,
) *Diagnostic {
	adt := q.QueryADT()
	cfg := q.QueryConfig()
	rp := q.QueryReadPattern()

	currentCost := qa.Cost.EstimatedLatencyMs
	if currentCost <= 0 {
		return nil
	}

	var bestAltName string
	var bestAltCost float64

	for _, eng := range engines {
		profile := eng.Profile()
		if profile.Name == qa.EngineName {
			continue
		}

		c, ok := profile.SupportsADT(adt)
		if !ok {
			continue
		}

		readC := effectiveReadComplexity(rp, c)
		cost := estimateCost(readC, cfg.Volume, profile.NsForRead(rp), profile.NetworkRTT)

		if bestAltName == "" || cost.EstimatedLatencyMs < bestAltCost {
			bestAltName = profile.Name
			bestAltCost = cost.EstimatedLatencyMs
		}
	}

	if bestAltName == "" {
		return nil
	}

	improvement := currentCost - bestAltCost
	if improvement <= 0 {
		return nil
	}

	fraction := improvement / currentCost
	if fraction <= DefaultRoutingHysteresis {
		return nil
	}

	return &Diagnostic{
		Level: DiagLevelInfo,
		Query: qa.QueryName,
		Message: fmt.Sprintf(
			"REPLAN-SUGGESTED: engine %q is %.0f%% cheaper (%.3fms vs %.3fms) than current %q"+
				" — live RTT shift detected; call Store.Replan to re-route",
			bestAltName, fraction*100, bestAltCost, currentCost, qa.EngineName,
		),
	}
}

// autoReplanLoop runs a background loop that periodically calls CheckRouting
// and, when diagnostics are emitted, calls Replan. This is the "set and forget"
// path for keeping the plan in sync with live latency. Use Store.StartAutoReplan
// to start it.
func (s *Store) autoReplanLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			diags := s.CheckRouting(ctx)
			if len(diags) > 0 {
				_ = s.Replan(ctx)
			}
		}
	}
}

// StartAutoReplan launches a background goroutine that periodically checks
// whether live latency shifts make a different engine cheaper and, when so,
// re-plans automatically. The interval controls how often the check runs
// (default 30s if zero). Cancelling the context stops the loop. The returned
// function can be called to stop the loop early (it cancels the context).
//
// This is the convenience path for long-lived Stores with ProbeEngine running.
// For finer control, use CheckRouting + Replan manually.
func (s *Store) StartAutoReplan(interval time.Duration) (stop func()) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	go s.autoReplanLoop(ctx, interval)

	return cancel
}
