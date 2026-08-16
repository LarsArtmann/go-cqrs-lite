package adttest

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// KnownGaps documents an ADT that violates a conformance rule with the reason
// it is currently acceptable, forcing explicit tracking instead of silence.
// Rule 1 gaps (declared native, not implemented) should reference a tracked
// backlog item; rule 2 gaps (implemented, undeclared) should explain why
// the planner deliberately does not route to the engine for that ADT.
//
// Alias of metaengine.CapabilityGaps: the audit core lives in the metaengine
// package so Doctor can render it (adttest imports metaengine, not the other
// way around).
type KnownGaps = metaengine.CapabilityGaps

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
// declaring ADTStreamLog (structural routing).
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

	res := metaengine.CapabilityAudit(engineName, eng, gaps)

	for _, line := range res.Table {
		t.Log(line)
	}

	for _, note := range res.Notes {
		t.Log(note)
	}

	for _, v := range res.Violations {
		t.Error(v)
	}
}

// CapabilityTable renders the declared-vs-implemented table for an engine as
// lines of text. Useful in status reports and ad-hoc audits; contains the
// same information RunCapabilityConformance logs.
func CapabilityTable(engineName string, eng metaengine.Engine) []string {
	return metaengine.CapabilityAudit(engineName, eng, nil).Table
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
	res := metaengine.CapabilityAudit(engineName, eng, gaps)

	return res.Table, res.Violations
}
