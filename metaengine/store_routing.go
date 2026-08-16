package metaengine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// DefaultRoutingHysteresis is the minimum fractional cost improvement required
// before CheckRouting suggests re-routing a query to a different engine. 20%
// means an alternative engine must be at least 20% cheaper than the current
// assignment before a REPLAN-SUGGESTED diagnostic is emitted. This deadband
// prevents oscillation when live RTT measurements jitter around the tie point.
// Override per-Store with WithRoutingHysteresis.
const DefaultRoutingHysteresis = 0.20

// DefaultRoutingMinDelta is the minimum absolute cost improvement (in
// milliseconds) required before CheckRouting suggests re-routing. This floor
// prevents re-routing on tiny absolute differences for very cheap queries
// (e.g. two local engines at 0.01ms), where a 20% fractional improvement is
// negligible. Override per-Store with WithRoutingMinDelta.
const DefaultRoutingMinDelta = 0.5

func defaultRoutingHysteresis(v float64) float64 {
	if v > 0 {
		return v
	}
	return DefaultRoutingHysteresis
}

func defaultRoutingMinDelta(v float64) float64 {
	if v > 0 {
		return v
	}
	return DefaultRoutingMinDelta
}

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

	// Differential: if no engine's RTT changed since the last check, the result
	// is identical — return the cached diagnostics without re-scoring.
	sig := s.routingSignature()

	s.routingMu.Lock()
	if sig == s.routingSig {
		cached := s.routingDiags
		s.routingMu.Unlock()
		return cached
	}
	s.routingMu.Unlock()

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

		diag := checkQueryRouting(q, qa, s.routableLocked(), s.routingHysteresis, s.routingMinDelta)
		if diag != nil {
			diags = append(diags, *diag)
		}
	}

	// Cache the result with the current signature.
	s.routingMu.Lock()
	s.routingSig = sig
	s.routingDiags = diags
	s.routingMu.Unlock()

	if len(diags) > 0 {
		slog.Info("metaengine: routing drift detected",
			"drift_count", len(diags), "queries", len(s.plan.Queries))
	}

	return diags
}

// routingSignature computes a string fingerprint of all engines' current
// NetworkRTT values. If this signature hasn't changed since the last
// CheckRouting call, the routing result is identical and can be served from
// cache. Only NetworkRTT varies at runtime (via live trackers); all other
// profile fields are static after construction.
func (s *Store) routingSignature() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder
	for _, eng := range s.engines {
		p := eng.Profile()
		fmt.Fprintf(&sb, "%s=%d|", p.Name, p.NetworkRTT.Nanoseconds())
	}

	return sb.String()
}

// checkQueryRouting re-scores all eligible engines for a single query and
// returns a REPLAN-SUGGESTED diagnostic if a cheaper alternative exists beyond
// the hysteresis deadband. Both the current assignment and alternatives are
// re-scored from their CURRENT Profile() (which reflects live tracker EWMA) —
// not the plan-time cost, which may be stale after a latency shift.
// Returns nil when the current assignment is still optimal (or near-optimal
// within the deadband).
func checkQueryRouting(
	q queryMeta,
	qa QueryAssignment,
	engines []Engine,
	hysteresis float64,
	minDelta float64,
) *Diagnostic {
	adt := q.QueryADT()
	cfg := q.QueryConfig()
	rp := q.QueryReadPattern()

	var currentCost float64
	found := false

	var bestAltName string
	var bestAltCost float64

	for _, eng := range engines {
		profile := eng.Profile()

		c, ok := profile.SupportsADT(adt)
		if !ok {
			continue
		}

		readC := effectiveReadComplexity(rp, c)
		cost := estimateCost(
			readC,
			cfg.Volume,
			profile.NsForRead(rp),
			profile.NetworkRTT,
		)

		if profile.Name == qa.EngineName {
			currentCost = cost.EstimatedLatencyMs
			found = true
		} else if bestAltName == "" || cost.EstimatedLatencyMs < bestAltCost {
			bestAltName = profile.Name
			bestAltCost = cost.EstimatedLatencyMs
		}
	}

	if !found || bestAltName == "" || currentCost <= 0 {
		return nil
	}

	improvement := currentCost - bestAltCost
	if improvement <= 0 {
		return nil
	}

	fraction := improvement / currentCost
	if fraction <= hysteresis {
		return nil
	}

	// Absolute floor: skip when the improvement is tiny in absolute terms,
	// even if the fraction is large (e.g. 50% of 0.01ms = 0.005ms).
	if improvement < minDelta {
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
				_ = s.replanWithTrigger(ctx, triggerAutoReroute)
			}
		}
	}
}

// StartAutoReplan launches a background goroutine that periodically checks
// whether live latency shifts make a different engine cheaper and, when so,
// re-plans automatically. The interval controls how often the check runs
// (default 30s if zero). Cancelling ctx stops the loop. The returned function
// can be called to stop the loop early (it cancels a child context derived from
// ctx).
//
// Pass a parent context so the goroutine's lifetime is tied to the caller's
// context tree — when the parent is cancelled, the loop stops:
//
//	autoStop := store.StartAutoReplan(ctx, 30*time.Second)
//	defer autoStop()
//
// This is the convenience path for long-lived Stores with ProbeEngine running.
// For finer control, use CheckRouting + Replan manually.
func (s *Store) StartAutoReplan(ctx context.Context, interval time.Duration) (stop func()) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(ctx)

	go s.autoReplanLoop(ctx, interval)

	return cancel
}
