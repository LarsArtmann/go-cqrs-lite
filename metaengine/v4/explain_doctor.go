package metaengine

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// The writeDoctor*Section helpers render the inline sections of
// (*Store).Doctor (explain.go). Each writes the SAME bytes the monolithic
// Doctor wrote before the maintidx refactor — order, spacing, and wording
// are load-bearing (sqliteengine/doctor_test.go pins sections).

func (s *Store) writeDoctorHealthSection(ctx context.Context, b *strings.Builder) {
	if err := s.HealthCheck(ctx); err != nil {
		fmt.Fprintf(b, "  UNHEALTHY: %v\n", err)
	} else {
		b.WriteString("  all engines healthy\n")
	}
}

func (s *Store) writeDoctorCollectionsSection(ctx context.Context, b *strings.Builder) {
	stats, err := s.Stats(ctx)
	if err != nil {
		fmt.Fprintf(b, "  ERROR: %v\n", err)

		return
	}

	for _, stat := range stats {
		rowStr := "N/A"
		if stat.RowCount >= 0 {
			rowStr = strconv.FormatInt(stat.RowCount, 10)
		}

		fmt.Fprintf(b, "  %s: %s rows (%s)\n", stat.Name, rowStr, stat.EngineName)
	}
}

func (s *Store) writeDoctorPoisonedSection(b *strings.Builder) {
	poisonedAny := false

	s.poison.mu.RLock()
	for col, err := range s.poison.m {
		fmt.Fprintf(b, "  %s: %v\n", col, err)
		poisonedAny = true
	}
	s.poison.mu.RUnlock()

	if !poisonedAny {
		b.WriteString("  none\n")
	}
}

// writeDoctorDurabilitySection renders the durability tiers and returns the
// engine snapshot it took — Doctor reuses it for the vector section so both
// sections observe ONE consistent snapshot (the pre-refactor behavior).
func (s *Store) writeDoctorDurabilitySection(b *strings.Builder) []Engine {
	s.mu.RLock() // art-dupl:accept idiomatic engine-snapshot prologue twin of Reset; extracting a helper for 3 lines costs more than the clone
	engines := slices.Clone(s.engines)
	s.mu.RUnlock()

	reported := false
	for _, eng := range engines {
		dr, ok := eng.(DurabilityReporter)
		if !ok {
			continue
		}

		tier := dr.EffectiveDurability()
		if tier == "" {
			tier = "engine-default"
		}

		fmt.Fprintf(b, "  %s: %s\n", eng.Profile().Name, tier)
		reported = true
	}

	if !reported {
		b.WriteString("  no engine reports an effective durability tier\n")
	}

	return engines
}

func (s *Store) writeDoctorReplicationSection(b *strings.Builder) {
	replicatedAny := false
	for _, c := range s.Collections() {
		if c.Replication == ReplicationNone {
			continue
		}

		fmt.Fprintf(b, "  %s: %s (lag=%dms, rtt=%dms)\n",
			c.Name, c.Replication, c.ReplicationLagMs, c.NetworkRTTMs)
		replicatedAny = true
	}

	if !replicatedAny {
		b.WriteString("  none\n")
	}
}

func (s *Store) writeDoctorPersistenceSection(b *strings.Builder) {
	volatileAny := false
	for _, c := range s.Collections() {
		if c.Persistence != PersistenceVolatile {
			continue
		}

		fmt.Fprintf(b, "  %s: volatile (engine=%s)\n", c.Name, c.EngineName)
		volatileAny = true
	}

	if !volatileAny {
		b.WriteString("  all persistent\n")
	}
}

func (s *Store) writeDoctorAggregateSection(b *strings.Builder) {
	aggAny := false
	s.mu.RLock()
	for _, name := range slices.Sorted(maps.Keys(s.queries)) {
		eng := s.queries[name].QueryEngine()
		caps := aggregateCapabilities(eng)
		if caps == "" {
			continue
		}

		fmt.Fprintf(b, "  %s: %s\n", name, caps)
		aggAny = true
	}
	s.mu.RUnlock()

	if !aggAny {
		b.WriteString("  none\n")
	}
}

func (s *Store) writeDoctorLatencySection(ctx context.Context, b *strings.Builder) {
	latAny := false
	for _, st := range s.GetEngineStats(ctx) {
		if !st.Profile.IsRemote() {
			continue
		}

		fmt.Fprintf(b, "  %s: %s\n", st.Name, FormatLiveLatency(st))
		latAny = true
	}

	if !latAny {
		b.WriteString("  all engines local\n")
	}
}

func (s *Store) writeDoctorRoutingSection(ctx context.Context, b *strings.Builder) {
	s.mu.RLock()
	version := 0
	computedAt := time.Time{}
	if s.plan != nil {
		version = s.plan.Version
		computedAt = s.plan.ComputedAt
	}

	replanCount := s.replanCount
	lastReplan := s.lastReplanAt
	hysteresis := s.routingHysteresis
	s.mu.RUnlock()

	fmt.Fprintf(b, "  plan version: %d\n", version)
	if !computedAt.IsZero() {
		fmt.Fprintf(
			b,
			"  computed: %s\n",
			humanize.RelTime(computedAt, time.Now(), "ago", "from now"),
		)
	}

	if replanCount > 0 {
		fmt.Fprintf(
			b,
			"  replans: %d (last %s)\n",
			replanCount,
			humanize.RelTime(lastReplan, time.Now(), "ago", "from now"),
		)
	} else {
		b.WriteString("  replans: 0 (never)\n")
	}

	if hist := s.formatPlanAuditTrail(); hist != "" {
		fmt.Fprintf(b, "  audit: %s\n", hist)
	}

	fmt.Fprintf(b, "  hysteresis: %.0f%%\n", hysteresis*100)

	driftDiags := s.CheckRouting(ctx)
	if len(driftDiags) > 0 {
		fmt.Fprintf(b, "  drift: %d queries would benefit from re-routing\n", len(driftDiags))
		for _, d := range driftDiags {
			fmt.Fprintf(b, "    %s\n", d.Message)
		}
	} else {
		b.WriteString("  drift: none\n")
	}
}
