package metaengine

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// adtContract maps a plan-declared ADT to the engine interface that natively
// serves it. Capability auditing compares Profile() declarations against
// these structural implementations. ADTGraph uses the internal graphBackend
// dispatch contract from dispatch.go (ADR-0113).
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
var adtContracts = map[ADT]adtContract{ //nolint:gochecknoglobals // immutable lookup table
	ADTMap:       {backend: reflect.TypeFor[MapBackend](), declarationRequired: true},
	ADTSet:       {backend: reflect.TypeFor[SetBackend](), declarationRequired: true},
	ADTCounter:   {backend: reflect.TypeFor[CounterBackend](), declarationRequired: true},
	ADTMultimap:  {backend: reflect.TypeFor[MultimapBackend](), declarationRequired: true},
	ADTLog:       {backend: reflect.TypeFor[LogBackend](), declarationRequired: true},
	ADTSortedMap: {backend: reflect.TypeFor[ScanBackend](), declarationRequired: true},
	ADTGraph:     {backend: reflect.TypeFor[graphBackend](), declarationRequired: true},
	ADTVector:    {backend: reflect.TypeFor[VectorBackend](), declarationRequired: true},
	ADTSearch:    {backend: reflect.TypeFor[SearchBackend](), declarationRequired: true},
	ADTSpatial:   {backend: reflect.TypeFor[SpatialBackend](), declarationRequired: true},
	// StreamLog is routed structurally (StreamLogBackend type assertions in
	// system adapters), not via Supports; declaring it is not yet required.
	ADTStreamLog: {backend: reflect.TypeFor[StreamLogBackend](), declarationRequired: false},
}

// CapabilityGaps documents an ADT that violates a conformance rule with the
// reason it is currently acceptable, forcing explicit tracking instead of
// silence. Rule 1 gaps (declared native, not implemented) should reference a
// tracked backlog item; rule 2 gaps (implemented, undeclared) should explain
// why the planner deliberately does not route to the engine for that ADT.
type CapabilityGaps map[ADT]string

// CapabilityAuditResult is the outcome of a declared-vs-implemented audit of
// one engine: the rendered table, the rule violations (empty when the engine
// conforms), and informational notes (never failures).
type CapabilityAuditResult struct {
	Table      []string
	Violations []string
	Notes      []string
}

// CapabilityAudit verifies an engine's Profile() declarations against its
// actual method surface. It is structural: behavioral parity stays in the
// adttest matrix. Rules:
//
//   - Rule 1 (over-declaration): ADT declared with native complexity (in
//     Supports, NOT in DegradedADTs) but the engine does not implement the
//     ADT's backend interface → violation unless documented in gaps.
//   - Rule 2 (under-declaration): backend implemented but the ADT is missing
//     from Supports (for contracts with declarationRequired) → violation
//     unless documented in gaps. The planner cannot route what is undeclared.
//   - Rule 3 (consistency): every DegradedADTs entry must also be in
//     Supports → violation unconditionally.
//
// Doctor renders this per engine in its "--- Capability ---" section; the
// adttest package exposes it as a test gate (RunCapabilityConformance).
func CapabilityAudit(engineName string, eng Engine, gaps CapabilityGaps) CapabilityAuditResult {
	profile := eng.Profile()

	res := CapabilityAuditResult{
		Table: []string{
			fmt.Sprintf("capability conformance: %s (%T)", engineName, eng),
			"ADT         | declared | implemented | verdict",
		},
	}

	for _, adt := range slices.Sorted(maps.Keys(adtContracts)) {
		contract := adtContracts[adt]

		implemented := contract.backend != nil &&
			reflect.TypeOf(eng).Implements(contract.backend)

		complexity, declared := profile.Supports[adt]

		row, violation := auditADTRow(engineName, adt, contract, gaps,
			complexity, declared, profile.DegradedADTs[adt], implemented)
		res.Table = append(res.Table, row)

		if violation != "" {
			res.Violations = append(res.Violations, violation)
		}
	}

	degradedViolations, notes := degradedSubsetViolations(engineName, profile)
	res.Violations = append(res.Violations, degradedViolations...)
	res.Notes = notes

	return res
}

// auditADTRow renders one table row plus the rule 1/2 violation ("" when the
// row conforms or the gap is documented). Informational statuses (upgrade
// candidates) are never violations.
func auditADTRow(
	engineName string,
	adt ADT,
	contract adtContract,
	gaps CapabilityGaps,
	complexity Complexity,
	declared, degraded, implemented bool,
) (row, violation string) {
	status := "ok"

	switch {
	case declared && !degraded && !implemented:
		status = "OVER-DECLARED (native claim, no backend)"

		if reason, ok := gaps[adt]; ok {
			status = "KNOWN GAP: " + reason
		} else {
			violation = fmt.Sprintf(
				"%s: ADT %s declared native (%s) but engine does not implement %s — "+
					"mark it degraded, implement it, or document it in CapabilityGaps",
				engineName, adt, complexity, contract.backend)
		}
	case implemented && contract.declarationRequired && !declared:
		status = "UNDER-DECLARED (backend, not in Supports)"

		if reason, ok := gaps[adt]; ok {
			status = "KNOWN GAP: " + reason
		} else {
			violation = fmt.Sprintf(
				"%s: engine implements %s for ADT %s but Profile().Supports does not "+
					"declare it — the planner cannot route to this engine; declare it "+
					"or document it in CapabilityGaps",
				engineName, contract.backend, adt)
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

	row = fmt.Sprintf("%-11s | %-8s | %-11s | %s", string(adt), declCell, implCell, status)

	return row, violation
}

// capabilityDoctorSection renders the "--- Capability ---" section of the
// Doctor() report: one conformance line per registered engine, with full
// violations inline. This surfaces lying engines at runtime, complementing
// the plan-time ADT diagnostics.
func (s *Store) capabilityDoctorSection() string {
	s.mu.RLock()
	engines := s.engines
	s.mu.RUnlock()

	var b strings.Builder

	b.WriteString("\n--- Capability ---\n")

	if len(engines) == 0 {
		b.WriteString("  no engines\n")

		return b.String()
	}

	for _, eng := range engines {
		name := eng.Profile().Name
		res := CapabilityAudit(name, eng, nil)

		if len(res.Violations) == 0 {
			fmt.Fprintf(&b, "  %s: declarations consistent\n", name)

			continue
		}

		fmt.Fprintf(&b, "  %s: %d conformance violation(s)\n", name, len(res.Violations))

		for _, v := range res.Violations {
			fmt.Fprintf(&b, "    %s\n", v)
		}
	}

	return b.String()
}

// degradedSubsetViolations enforces rule 3 plus the degraded-ADTs note.
func degradedSubsetViolations(
	engineName string,
	profile EngineProfile,
) (violations, notes []string) {
	for adt := range profile.DegradedADTs {
		if _, declared := profile.Supports[adt]; !declared {
			violations = append(violations, fmt.Sprintf(
				"%s: ADT %s is in DegradedADTs but not in Supports — an engine cannot "+
					"degrade an ADT it does not declare", engineName, adt))
		}
	}

	if len(profile.DegradedADTs) > 0 {
		texts := make([]string, 0, len(profile.DegradedADTs))

		for _, adtName := range slices.Sorted(maps.Keys(profile.DegradedADTs)) {
			texts = append(texts, string(adtName))
		}

		notes = append(notes, fmt.Sprintf("%s: degraded ADTs: %s",
			engineName, strings.Join(texts, ", ")))
	}

	return violations, notes
}
