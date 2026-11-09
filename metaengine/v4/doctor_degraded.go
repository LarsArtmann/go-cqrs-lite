package metaengine

import (
	"fmt"
	"strings"
)

// degradedDoctorSection renders the "--- Degraded ADTs ---" section of the
// Doctor() report. It lists every query routed to an engine that handles its
// ADT via a degraded (brute-force) fallback, including the estimated latency
// penalty and a native-engine recommendation if one is available.
//
// This gives operators a runtime view of which queries are running on
// suboptimal backends, complementing the plan-time DEGRADED diagnostic
// emitted by degradedADTRule.
func (s *Store) degradedDoctorSection() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.plan == nil {
		return ""
	}

	var lines []string

	for _, q := range s.plan.Queries {
		meta, ok := s.queries[q.QueryName]
		if !ok {
			continue
		}

		profile := meta.QueryEngine().Profile()
		if !profile.IsDegraded(q.ADT) {
			continue
		}

		recommendation := findNativeADTEngine(
			s.engines, q.ADT, profile.Name,
		)

		line := fmt.Sprintf(
			"  %s: %s via %s (%s fallback) est %.1fms",
			q.QueryName, q.ADT, profile.Name,
			q.Complexity, q.Cost.EstimatedLatencyMs,
		)

		if recommendation != "" {
			line += fmt.Sprintf(" — native %q available", recommendation)
		} else {
			line += " — no native engine"
		}

		lines = append(lines, line)
	}

	var b strings.Builder
	b.WriteString("\n--- Degraded ADTs ---\n")

	if len(lines) == 0 {
		b.WriteString("  none\n")
	} else {
		for _, l := range lines {
			fmt.Fprintln(&b, l)
		}
	}

	return b.String()
}
