package metaengine

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// DefaultEngineFailureThreshold is the number of consecutive classified
// failures (errorfamily Infrastructure or Transient) that quarantines an
// engine (ADR-0137). Override per-Store with SetEngineFailureThreshold.
const DefaultEngineFailureThreshold = 3

// DefaultReprobeInterval is the polling period StartAutoReprobe uses when
// given a non-positive interval.
const DefaultReprobeInterval = 10 * time.Second

// reprobeTimeout caps a single health probe so a hung engine cannot stall
// the reprobe loop.
const reprobeTimeout = 5 * time.Second

// EngineHealthState is an engine's deactivation state (ADR-0137).
type EngineHealthState string

const (
	// EngineActive is the normal state: the engine serves reads and writes.
	EngineActive EngineHealthState = "active"
	// EngineQuarantined means the engine failed past the threshold; reads
	// reroute to the next-best capable engine until reactivation.
	EngineQuarantined EngineHealthState = "quarantined"
)

// EngineHealth is the observable per-engine health snapshot surfaced by
// HealthSnapshot, GetEngineStats, and Doctor.
type EngineHealth struct {
	State               EngineHealthState
	ConsecutiveFailures int
	QuarantinedAt       time.Time // zero when active
	LastError           string    // last classified failure; empty when none
}

type engineHealthRecord struct {
	fails       int
	quarantined bool
	at          time.Time
	lastErr     string
}

// SetEngineFailureThreshold sets how many consecutive classified failures
// quarantine an engine. Values below 1 are ignored.
func (s *Store) SetEngineFailureThreshold(n int) {
	if n < 1 {
		return
	}

	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	s.engineFailureThreshold = n
}

// effectiveFailureThreshold reads the threshold with healthMu already held.
func (s *Store) effectiveFailureThreshold() int {
	if s.engineFailureThreshold > 0 {
		return s.engineFailureThreshold
	}

	return DefaultEngineFailureThreshold
}

// recordEngineFailure classifies err and, when it is an Infrastructure or
// Transient failure (the "backend unavailable/overloaded" families), counts
// it against the engine. Reaching the threshold quarantines the engine.
// Rejection, Conflict, Corruption, and unclassified errors never count —
// client mistakes and data corruption must fail loudly, not trigger failover.
func (s *Store) recordEngineFailure(eng Engine, err error) {
	if eng == nil || err == nil {
		return
	}

	switch errorfamily.Classify(err) {
	case errorfamily.Infrastructure, errorfamily.Transient:
	default:
		return
	}

	name := eng.Profile().Name

	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	if s.health == nil {
		s.health = make(map[string]*engineHealthRecord)
	}

	rec := s.health[name]
	if rec == nil {
		rec = &engineHealthRecord{}
		s.health[name] = rec
	}

	rec.fails++
	rec.lastErr = err.Error()

	if !rec.quarantined && rec.fails >= s.effectiveFailureThreshold() {
		rec.quarantined = true
		rec.at = time.Now()

		slog.Warn("metaengine: engine quarantined after consecutive failures",
			"engine", name, "failures", rec.fails, "last_error", rec.lastErr)
	}
}

// recordEngineSuccess resets the consecutive-failure counter. It does NOT
// lift a quarantine — reactivation requires a successful probe
// (StartAutoReprobe) or an explicit ReactivateEngine.
func (s *Store) recordEngineSuccess(eng Engine) {
	if eng == nil {
		return
	}

	name := eng.Profile().Name

	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	if rec := s.health[name]; rec != nil {
		rec.fails = 0
	}
}

// engineQuarantined reports the engine's deactivation state by name.
func (s *Store) engineQuarantined(name string) bool {
	s.healthMu.RLock()
	defer s.healthMu.RUnlock()

	return s.engineQuarantinedLocked(name)
}

// engineQuarantinedLocked reads quarantine state with healthMu already held.
func (s *Store) engineQuarantinedLocked(name string) bool {
	rec := s.health[name]

	return rec != nil && rec.quarantined
}

// HealthSnapshot returns the current per-engine health state keyed by name.
func (s *Store) HealthSnapshot() map[string]EngineHealth {
	s.healthMu.RLock()
	defer s.healthMu.RUnlock()

	out := make(map[string]EngineHealth, len(s.health))
	for name, rec := range s.health {
		out[name] = engineHealthOf(rec)
	}

	return out
}

func engineHealthOf(rec *engineHealthRecord) EngineHealth {
	state := EngineActive
	if rec.quarantined {
		state = EngineQuarantined
	}

	return EngineHealth{
		State:               state,
		ConsecutiveFailures: rec.fails,
		QuarantinedAt:       rec.at,
		LastError:           rec.lastErr,
	}
}

// ReactivateEngine manually lifts a quarantine. Returns true when the engine
// was quarantined. Engines without a Prober can only return this way.
func (s *Store) ReactivateEngine(name string) bool {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()

	rec := s.health[name]
	if rec == nil || !rec.quarantined {
		return false
	}

	rec.quarantined = false
	rec.fails = 0
	rec.at = time.Time{}
	rec.lastErr = ""

	slog.Info("metaengine: engine reactivated", "engine", name)

	return true
}

// doctorEngineHealthSection renders the ADR-0137 per-engine health lines for
// the Doctor report: quarantine state, consecutive failures, and the last
// classified error, sorted by engine name.
func (s *Store) doctorEngineHealthSection() string {
	health := s.HealthSnapshot()
	if len(health) == 0 {
		return "  no failures recorded\n"
	}

	names := slices.Sorted(maps.Keys(health))

	var b strings.Builder

	for _, name := range names {
		h := health[name]

		switch {
		case h.State == EngineQuarantined:
			fmt.Fprintf(&b, "  %s: QUARANTINED (failures=%d, since=%s, last=%q)\n",
				name, h.ConsecutiveFailures, h.QuarantinedAt.Format(time.RFC3339), h.LastError)

		case h.ConsecutiveFailures > 0:
			fmt.Fprintf(
				&b,
				"  %s: active (failures=%d, last=%q)\n",
				name,
				h.ConsecutiveFailures,
				h.LastError,
			)

		default:
			fmt.Fprintf(&b, "  %s: active\n", name)
		}
	}

	return b.String()
}

// routedQuery overrides one query's engine for an execution-scoped reroute
// (ADR-0137). Unlike shadowQuery it keeps watcher-notification semantics:
// reads never notify, and the plan is untouched.
type routedQuery struct {
	queryMeta

	eng Engine
}

func (r routedQuery) QueryEngine() Engine { return r.eng }

// effectiveQueryLocked returns the query to execute: the planned one, or a
// health-rerouted variant when its assigned engine is quarantined and a
// cheaper non-quarantined engine natively serves the query's ADT. Callers
// must hold s.mu (read or write); the same capability-aware partition rule
// as the planner applies, so a reroute target is one the planner itself
// could have chosen. With no candidate the planned query is returned and the
// original engine error surfaces to the caller.
func (s *Store) effectiveQueryLocked(q queryMeta) queryMeta {
	assigned := q.QueryEngine()
	if assigned == nil || !s.engineQuarantined(assigned.Profile().Name) {
		return q
	}

	alt := s.bestHealthyEngineLocked(q)
	if alt == nil {
		return q
	}

	slog.Warn("metaengine: rerouting query around quarantined engine",
		"query", q.QueryName(), "from", assigned.Profile().Name, "to", alt.Profile().Name)

	return routedQuery{queryMeta: q, eng: alt}
}

// bestHealthyEngineLocked scores every non-quarantined, routable engine that
// natively serves the query's ADT and returns the cheapest. Callers must hold
// s.mu; returns nil when no candidate exists.
func (s *Store) bestHealthyEngineLocked(q queryMeta) Engine {
	adt := q.QueryADT()
	cfg := q.QueryConfig()
	rp := q.QueryReadPattern()
	assignedName := q.QueryEngine().Profile().Name

	var best Engine
	var bestCost float64

	for _, eng := range s.routableLocked() {
		profile := eng.Profile()

		if profile.Name == assignedName || s.engineQuarantined(profile.Name) {
			continue
		}

		c, ok := profile.SupportsADT(adt)
		if !ok || !engineServesADTNatively(eng, adt) {
			continue
		}

		cost := estimateCost(
			effectiveReadComplexity(rp, c),
			cfg.Volume,
			profile.NsForRead(rp),
			profile.NetworkRTT,
		).EstimatedLatencyMs

		if best == nil || cost < bestCost {
			best = eng
			bestCost = cost
		}
	}

	return best
}

// StartAutoReprobe polls quarantined engines that implement Prober and
// reactivates those whose probe succeeds. Engines without a Prober stay
// quarantined until ReactivateEngine — an engine we cannot probe is an
// engine we cannot vouch for. The returned stop func cancels the loop and
// waits for it to exit.
func (s *Store) StartAutoReprobe(ctx context.Context, interval time.Duration) (stop func()) {
	if interval <= 0 {
		interval = DefaultReprobeInterval
	}

	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				s.reprobeOnce(ctx)
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

// reprobeOnce probes every quarantined engine with Prober capability and
// reactivates the ones that answer. Engines are snapshotted before probing
// so a hung backend cannot hold the Store lock.
func (s *Store) reprobeOnce(ctx context.Context) {
	type candidate struct {
		name string
		eng  Engine
	}

	var candidates []candidate

	s.mu.RLock()
	for _, eng := range s.engines {
		if _, ok := eng.(Prober); !ok {
			continue
		}

		name := eng.Profile().Name
		if s.engineQuarantined(name) {
			candidates = append(candidates, candidate{name: name, eng: eng})
		}
	}
	s.mu.RUnlock()

	for _, c := range candidates {
		probeCtx, cancel := context.WithTimeout(ctx, reprobeTimeout)

		prober := c.eng.(Prober) //nolint:forcetypeassert // capability checked under lock above

		_, err := prober.Probe(probeCtx)
		cancel()

		if err == nil {
			s.ReactivateEngine(c.name)
		}
	}
}
