package adttest

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// adtContract maps a plan-declared ADT to the engine interface that natively
// serves it. The conformance rules compare Profile() declarations against
// these structural implementations.
type adtContract struct {
	// backend is the interface whose methods natively implement the ADT.
	backend reflect.Type
	// declarationRequired: when an engine implements backend, the ADT must
	// appear in Profile().Supports so the planner can route to it. False for
	// ADTs routed structurally today (StreamLog).
	declarationRequired bool
}

// adtContracts is the declared-vs-implemented table. ADTSortedMap has no
// dedicated backend: it is served by MapBackend writes + ScanBackend ordered
// reads (pushdown), so ScanBackend is its structural marker.
//
// GraphBackend here is the local dispatch contract (GraphAddEdge /
// GraphNeighbors) mirroring metaengine's per-engine graph methods; ADR-0113
// keeps the public GraphDriver in the graph module.
var adtContracts = map[metaengine.ADT]adtContract{ //nolint:gochecknoglobals // immutable lookup table
	metaengine.ADTMap: {
		backend:             reflect.TypeFor[metaengine.MapBackend](),
		declarationRequired: true,
	},
	metaengine.ADTSet: {
		backend:             reflect.TypeFor[metaengine.SetBackend](),
		declarationRequired: true,
	},
	metaengine.ADTCounter: {
		backend:             reflect.TypeFor[metaengine.CounterBackend](),
		declarationRequired: true,
	},
	metaengine.ADTMultimap: {
		backend:             reflect.TypeFor[metaengine.MultimapBackend](),
		declarationRequired: true,
	},
	metaengine.ADTLog: {
		backend:             reflect.TypeFor[metaengine.LogBackend](),
		declarationRequired: true,
	},
	metaengine.ADTSortedMap: {
		backend:             reflect.TypeFor[metaengine.ScanBackend](),
		declarationRequired: true,
	},
	metaengine.ADTGraph: {backend: reflect.TypeFor[graphBackend](), declarationRequired: true},
	metaengine.ADTVector: {
		backend:             reflect.TypeFor[metaengine.VectorBackend](),
		declarationRequired: true,
	},
	metaengine.ADTSearch: {
		backend:             reflect.TypeFor[metaengine.SearchBackend](),
		declarationRequired: true,
	},
	metaengine.ADTSpatial: {
		backend:             reflect.TypeFor[metaengine.SpatialBackend](),
		declarationRequired: true,
	},
	// StreamLog is routed structurally (StreamLogBackend type assertions in
	// system adapters), not via Supports; declaring it is not yet required.
	metaengine.ADTStreamLog: {
		backend:             reflect.TypeFor[metaengine.StreamLogBackend](),
		declarationRequired: false,
	},
}

// KnownGaps documents an ADT that violates a conformance rule with the reason
// it is currently acceptable, forcing explicit tracking instead of silence.
// Rule 1 gaps (declared native, not implemented) should reference a tracked
// backlog item; rule 2 gaps (implemented, undeclared) should explain why
// the planner deliberately does not route to the engine for that ADT.
type KnownGaps map[metaengine.ADT]string

// RunCapabilityConformance verifies an engine's Profile() declarations
// against its actual method surface (the "declared vs implemented" table
// from the 2026-08-16 brutal review). It is structural: behavioral parity
// stays in RunMatrix. Rules:
//
//   - Rule 1 (over-declaration): ADT declared with native complexity (in
//     Supports, NOT in DegradedADTs) but the engine does not implement the
//     ADT's backend interface → FAIL unless documented in gaps.
//   - Rule 2 (under-declaration): backend implemented but the ADT is missing
//     from Supports (for contracts with declarationRequired) → FAIL unless
//     documented in gaps. The planner cannot route what is undeclared.
//   - Rule 3 (consistency): every DegradedADTs entry must also be in
//     Supports → FAIL unconditionally.
//
// Informational notes (never failures): declared degraded but natively
// implemented (upgrade candidate), and implemented StreamLogBackend without
// declaring ADTStreamLog (structural routing, documented above).
func RunCapabilityConformance(
	t *testing.T,
	engineName string,
	eng metaengine.Engine,
	gaps KnownGaps,
) {
	t.Helper()

	if eng == nil {
		t.Fatal("adttest.RunCapabilityConformance: nil engine")
	}

	table, violations, notes := capabilityTable(engineName, eng, gaps)

	for _, line := range table {
		t.Log(line)
	}

	for _, note := range notes {
		t.Log(note)
	}

	for _, v := range violations {
		t.Error(v)
	}
}

// CapabilityTable renders the declared-vs-implemented table for an engine as
// lines of text. Useful in status reports and ad-hoc audits; contains the
// same information RunCapabilityConformance logs.
func CapabilityTable(engineName string, eng metaengine.Engine) []string {
	table, _, _ := capabilityTable(engineName, eng, nil)

	return table
}

// AuditCapability returns the rendered capability table plus the rule
// violations for an engine. It is the plumbing-free form of
// RunCapabilityConformance: callers that want to assert or persist the
// findings (Diagnostics tooling, reports) use this instead of a testing.T.
func AuditCapability(
	engineName string,
	eng metaengine.Engine,
	gaps KnownGaps,
) (table, violations []string) {
	table, violations, _ = capabilityTable(engineName, eng, gaps)

	return table, violations
}

func capabilityTable(
	engineName string,
	eng metaengine.Engine,
	gaps KnownGaps,
) (table, violations, notes []string) {
	profile := eng.Profile()

	table = append(table,
		fmt.Sprintf("capability conformance: %s (%T)", engineName, eng),
		"ADT         | declared | implemented | verdict")

	adts := make([]string, 0, len(adtContracts))
	for adt := range adtContracts {
		adts = append(adts, string(adt))
	}

	sort.Strings(adts)

	for _, adtName := range adts {
		adt := metaengine.ADT(adtName)
		contract := adtContracts[adt]

		implemented := contract.backend != nil &&
			reflect.TypeOf(eng).Implements(contract.backend)

		complexity, declared := profile.Supports[adt]
		degraded := profile.DegradedADTs[adt]

		status := "ok"
		switch {
		case declared && !degraded && !implemented:
			status = "OVER-DECLARED (native claim, no backend)"
			if reason, ok := gaps[adt]; ok {
				status = "KNOWN GAP: " + reason
			} else {
				violations = append(violations, fmt.Sprintf(
					"%s: ADT %s declared native (%s) but engine does not implement %s — "+
						"mark it degraded, implement it, or document it in KnownGaps",
					engineName, adt, complexity, contract.backend))
			}
		case implemented && contract.declarationRequired && !declared:
			status = "UNDER-DECLARED (backend, not in Supports)"
			if reason, ok := gaps[adt]; ok {
				status = "KNOWN GAP: " + reason
			} else {
				violations = append(violations, fmt.Sprintf(
					"%s: engine implements %s for ADT %s but Profile().Supports does not "+
						"declare it — the planner cannot route to this engine; declare it "+
						"or document it in KnownGaps",
					engineName, contract.backend, adt))
			}
		case declared && degraded && implemented:
			status = "note: declared degraded but natively implemented (upgrade candidate)"
		}

		declCell := "—"
		if declared {
			declCell = string(complexity)
			if degraded {
				declCell += " (degraded)"
			}
		}

		implCell := "no"
		if implemented {
			implCell = "yes"
		}

		table = append(table, fmt.Sprintf("%-11s | %-8s | %-11s | %s",
			string(adt), declCell, implCell, status))
	}

	degradedViolations, notes := degradedSubsetViolations(engineName, profile)
	violations = append(violations, degradedViolations...)

	return table, violations, notes
}

func degradedSubsetViolations(
	engineName string,
	profile metaengine.EngineProfile,
) (violations, notes []string) {
	for adt := range profile.DegradedADTs {
		if _, declared := profile.Supports[adt]; !declared {
			violations = append(violations, fmt.Sprintf(
				"%s: ADT %s is in DegradedADTs but not in Supports — an engine cannot "+
					"degrade an ADT it does not declare", engineName, adt))
		}
	}

	if len(profile.DegradedADTs) > 0 {
		names := make([]string, 0, len(profile.DegradedADTs))
		for adt := range profile.DegradedADTs {
			names = append(names, string(adt))
		}

		sort.Strings(names)
		notes = append(notes, fmt.Sprintf("%s: degraded ADTs: %s",
			engineName, strings.Join(names, ", ")))
	}

	return violations, notes
}
