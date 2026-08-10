package metaengine

import (
	"context"
	"fmt"
	"time"
)

// EngineStats is a per-engine runtime measurement view surfaced by
// GetEngineStats. It pairs the profile the planner uses with the live latency
// that backs it, so operators can see whether routing runs on measured or
// prior/stale values.
type EngineStats struct {
	Name string
	// Profile is what the planner currently sees from Profile().
	Profile EngineProfile

	// MeasuredRTT is the live RTT snapshot (EWMA + percentiles). Zero when no
	// RTT tracker is installed.
	MeasuredRTT LatencyStats
	// MeasuredRead is the live per-read latency snapshot. Zero when no read
	// tracker is installed.
	MeasuredRead LatencyStats

	// HasLiveRTT reports whether a live RTT tracker is installed at all.
	HasLiveRTT bool
	// HasLiveRead reports whether a live per-read tracker is installed.
	HasLiveRead bool

	// Samples is the RTT sample count (0 when no tracker).
	Samples int
	// LastProbe is the timestamp of the most recent RTT sample.
	LastProbe time.Time
	// Stale is true when the engine is remote (RequiresNetwork or a non-zero
	// prior RTT) but its measurement is missing or older than the stale-after
	// window. Routing on a stale value falls back to the declared prior — this
	// flag makes that fallback visible instead of silent.
	Stale bool
}

// GetEngineStats returns a live measurement report for every engine in the
// Store. It re-reads each engine's Profile() (so live tracker values are
// current) and pairs it with the live latency snapshot. Remote engines without
// fresh measurement are marked Stale. Use this for the Doctor report,
// readiness probes, and any operator dashboard that needs to show whether the
// planner is routing on measured or prior latency.
func (s *Store) GetEngineStats(ctx context.Context) []EngineStats {
	s.mu.RLock()
	engines := s.engines
	s.mu.RUnlock()

	out := make([]EngineStats, 0, len(engines))

	for _, eng := range engines {
		out = append(out, buildEngineStats(eng))
	}

	return out
}

// liveLatencyReporter is satisfied by *Calibration (and thus every engine that
// embeds it). Kept as a named interface so custom engines can implement it too.
type liveLatencyReporter interface {
	LiveLatency() LiveLatency
}

func buildEngineStats(eng Engine) EngineStats {
	profile := eng.Profile()

	stats := EngineStats{
		Name:    profile.Name,
		Profile: profile,
	}

	rttFresh := false
	if reporter, ok := eng.(liveLatencyReporter); ok {
		live := reporter.LiveLatency()
		stats.HasLiveRTT = live.HasRTT
		stats.HasLiveRead = live.HasRead
		stats.MeasuredRTT = live.RTT
		stats.MeasuredRead = live.Read
		stats.Samples = live.RTT.Samples
		stats.LastProbe = live.RTT.LastSample
		rttFresh = live.Fresh
	}

	// A remote engine is stale when its RTT measurement is missing or not
	// current. We use the tracker's own Fresh() determination (authoritative:
	// it knows the configured stale-after window) instead of a display-side
	// approximation. Local engines are never stale — RTT is structurally zero.
	if profile.IsRemote() && !rttFresh {
		stats.Stale = true
	}

	return stats
}

// FormatLiveLatency renders a one-line live-latency summary for EXPLAIN/Doctor.
// Example outputs:
//
//	rtt=live 2.1ms (p95 4.0ms, n=512)
//	rtt=prior 1ms [stale, no live samples]
//	rtt=0s (local)
func FormatLiveLatency(st EngineStats) string {
	if !st.Profile.IsRemote() {
		return "rtt=0s (local)"
	}

	if st.HasLiveRTT && st.Samples > 0 && !st.Stale {
		return fmt.Sprintf("rtt=live %s (p95 %s, n=%d)",
			roundDur(st.MeasuredRTT.EWMA),
			roundDur(st.MeasuredRTT.P95),
			st.Samples,
		)
	}

	prior := st.Profile.NetworkRTT
	if st.Stale {
		if st.HasLiveRTT && st.Samples > 0 {
			return fmt.Sprintf("rtt=live %s [stale, last %s ago]",
				roundDur(st.MeasuredRTT.EWMA),
				roundDur(time.Since(st.LastProbe)),
			)
		}

		return fmt.Sprintf("rtt=prior %s [stale, no live samples]", roundDur(prior))
	}

	return fmt.Sprintf("rtt=prior %s", roundDur(prior))
}

func roundDur(d time.Duration) time.Duration {
	switch {
	case d >= time.Millisecond:
		return d.Round(100 * time.Microsecond)
	case d >= time.Microsecond:
		return d.Round(100 * time.Nanosecond)
	default:
		return d
	}
}
