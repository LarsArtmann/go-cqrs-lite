package metaengine

import (
	"fmt"
	"strings"
)

// resetDoctorSection renders the "--- Reset ---" section of the Doctor()
// report: which engines implement [EngineResetter] (the ADR-0136 capability
// ladder). Operators read this BEFORE calling [Store.Reset] — engines listed
// as not bulk-clearable keep stale materialized state across the reset, so a
// replay would fold on top of leftovers instead of rebuilding from zero.
func (s *Store) resetDoctorSection() string {
	//art-dupl:accept Doctor-report section preamble (engine snapshot, header, empty short-circuit) is the shared reporting idiom; each section's body is fully divergent
	engines := s.enginesSnapshot()

	var b strings.Builder

	b.WriteString("\n--- Reset ---\n")

	if len(engines) == 0 {
		b.WriteString("  no engines\n")

		return b.String()
	}

	unclearable := false

	for _, eng := range engines {
		name := eng.Profile().Name
		if name == "" {
			name = "engine"
		}

		if _, ok := eng.(EngineResetter); ok {
			fmt.Fprintf(&b, "  %s: reset-capable\n", name)

			continue
		}

		fmt.Fprintf(
			&b,
			"  %s: NOT bulk-clearable (no EngineResetter) — Store.Reset will be partial\n",
			name,
		)
		unclearable = true
	}

	if unclearable {
		b.WriteString(
			"  remedy: route those collections to a reset-capable engine or clear its data out-of-band before replaying\n",
		)
	}

	return b.String()
}
