package adttest_test

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// lyingEngine implements SetBackend but declares nothing, declares Vector
// natively without implementing VectorBackend, and degrades Log without
// declaring it — one violation of each conformance rule. The embedded nil
// Engine is never called: the audit only reads Profile() and the structural
// method surface.
type lyingEngine struct {
	metaengine.Engine // nil; intentional — the audit never invokes it

	profile metaengine.EngineProfile
}

func (e *lyingEngine) Profile() metaengine.EngineProfile { return e.profile }

func (e *lyingEngine) SetAdd(context.Context, string, any) error { return nil }

func (e *lyingEngine) SetContains(context.Context, string, any) (bool, error) { return false, nil }

func newLyingEngine() *lyingEngine {
	return &lyingEngine{
		profile: metaengine.EngineProfile{
			Name: "lying",
			Supports: map[metaengine.ADT]metaengine.Complexity{
				metaengine.ADTVector: metaengine.ComplexityON,
			},
			DegradedADTs: map[metaengine.ADT]bool{
				metaengine.ADTLog: true,
			},
		},
	}
}

// TestAuditCapability_RulesFire proves each rule produces its own violation
// line (guards the append-merge in capabilityTable: rule-1/2 violations must
// survive alongside the rule-3 violations from degradedSubsetViolations).
func TestAuditCapability_RulesFire(t *testing.T) {
	t.Parallel()

	_, violations := adttest.AuditCapability("lying", newLyingEngine(), nil)
	got := strings.Join(violations, "\n")

	if !strings.Contains(got, "ADT vector declared native") {
		t.Errorf("missing rule-1 violation (over-declared vector):\n%s", got)
	}

	if !strings.Contains(got, "implements metaengine.SetBackend for ADT set") {
		t.Errorf("missing rule-2 violation (under-declared set):\n%s", got)
	}

	if !strings.Contains(got, "ADT log is in DegradedADTs but not in Supports") {
		t.Errorf("missing rule-3 violation (degraded log undeclared):\n%s", got)
	}

	if n := len(violations); n != 3 {
		t.Errorf("got %d violations, want exactly 3:\n%s", n, got)
	}
}

// TestAuditCapability_KnownGapSuppresses proves a documented gap turns a
// violation into a table annotation instead of a failure.
func TestAuditCapability_KnownGapSuppresses(t *testing.T) {
	t.Parallel()

	table, violations := adttest.AuditCapability("lying", newLyingEngine(), adttest.KnownGaps{
		metaengine.ADTVector: "vector backend planned (tracked backlog item)",
	})

	got := strings.Join(violations, "\n")

	if strings.Contains(got, "ADT vector declared native") {
		t.Errorf("rule-1 violation must be suppressed by KnownGaps:\n%s", got)
	}

	if !strings.Contains(strings.Join(table, "\n"), "KNOWN GAP: vector backend planned") {
		t.Errorf("table must show the KNOWN GAP verdict:\n%s", strings.Join(table, "\n"))
	}

	if !strings.Contains(got, "implements metaengine.SetBackend for ADT set") {
		t.Errorf("undocumented rule-2 violation must still fire:\n%s", got)
	}
}

// TestAuditCapability_MemoryEngineIsClean is the negative control: a fully
// conformant engine produces zero violations.
func TestAuditCapability_MemoryEngineIsClean(t *testing.T) {
	t.Parallel()

	_, violations := adttest.AuditCapability("memory", metaengine.NewMemoryEngine(), nil)

	if len(violations) != 0 {
		t.Errorf("memory engine must be conformant, got:\n%s", strings.Join(violations, "\n"))
	}
}
