package metaengine_test

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// auditLyingEngine declares Vector natively without implementing
// VectorBackend, implements SetBackend without declaring ADTSet, degrades
// Log without declaring it, and stays silent on the write-side ADTs — one
// violation per conformance rule. The embedded nil Engine is never invoked:
// the audit only reads Profile() and the structural method surface.
type auditLyingEngine struct {
	metaengine.Engine // nil; intentional — the audit never invokes it

	profile metaengine.EngineProfile
}

func (e *auditLyingEngine) Profile() metaengine.EngineProfile { return e.profile }

func (e *auditLyingEngine) SetAdd(_ context.Context, _ string, _ any) error { return nil }

func (e *auditLyingEngine) SetContains(_ context.Context, _ string, _ any) (bool, error) {
	return false, nil
}

func newAuditLyingEngine() *auditLyingEngine {
	return &auditLyingEngine{
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

func TestCapabilityAudit_AllRulesFire(t *testing.T) {
	t.Parallel()

	res := metaengine.CapabilityAudit("lying", newAuditLyingEngine(), nil)

	if len(res.Violations) != 5 {
		t.Fatalf("expected 5 violations (one per rule, two for rule 4's silence), got %d: %v",
			len(res.Violations), res.Violations)
	}

	joined := strings.Join(res.Violations, "\n")

	for _, want := range []string{
		"ADT vector declared native",          // rule 1
		"planner cannot route to this engine", // rule 2
		"in DegradedADTs but not in Supports", // rule 3
		"does not implement",                  // rule 1 detail
		"neither supported nor refused",       // rule 4 (write-side silence)
		"never silence",                       // rule 4 detail
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("violations missing %q:\n%s", want, joined)
		}
	}

	if len(res.Table) != len(metaengine.CapabilityAudit("x", newAuditLyingEngine(), nil).Table) {
		t.Error("table rendering should be deterministic in length")
	}
}

func TestCapabilityAudit_MemoryEngineConforms(t *testing.T) {
	t.Parallel()

	res := metaengine.CapabilityAudit("memory", metaengine.NewMemoryEngine(), nil)

	if len(res.Violations) != 0 {
		t.Fatalf("memory engine should conform, got: %v", res.Violations)
	}
}

func TestCapabilityAudit_RefusalRules(t *testing.T) {
	t.Parallel()

	t.Run("disjointness: supported and refused is a violation", func(t *testing.T) {
		t.Parallel()

		eng := &auditLyingEngine{profile: metaengine.EngineProfile{
			Name: "both",
			Supports: map[metaengine.ADT]metaengine.Complexity{
				metaengine.ADTDueClaim: metaengine.ComplexityOLogN,
			},
			RefusedADTs: map[metaengine.ADT]string{
				metaengine.ADTDueClaim: "inconsistent",
			},
		}}

		res := metaengine.CapabilityAudit("both", eng, nil)

		joined := strings.Join(res.Violations, "\n")
		if !strings.Contains(joined, "both supported and refused") {
			t.Fatalf("expected disjointness violation, got: %v", res.Violations)
		}
	})

	t.Run("refusal renders in the table and is not a violation", func(t *testing.T) {
		t.Parallel()

		const reason = "leaderless replication cannot fence concurrent claimers"

		eng := &auditRefusingEngine{profile: metaengine.EngineProfile{
			Name: "refusing",
			RefusedADTs: map[metaengine.ADT]string{
				metaengine.ADTDueClaim: reason,
				metaengine.ADTDedup:    reason,
			},
		}}

		res := metaengine.CapabilityAudit("refusing", eng, nil)

		if len(res.Violations) != 0 {
			t.Fatalf("an honest refusal is not a violation, got: %v", res.Violations)
		}

		table := strings.Join(res.Table, "\n")
		if !strings.Contains(table, "REFUSED: "+reason) {
			t.Fatalf("refusal reason missing from audit table:\n%s", table)
		}
	})
}

// auditRefusingEngine implements NO backend and refuses the write-side ADTs
// with an architectural reason — the honest minimal engine.
type auditRefusingEngine struct {
	metaengine.Engine // nil; intentional — the audit never invokes it

	profile metaengine.EngineProfile
}

func (e *auditRefusingEngine) Profile() metaengine.EngineProfile { return e.profile }

func TestDoctor_CapabilitySection(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	output := store.Doctor(t.Context())

	for _, want := range []string{
		"--- Capability ---",
		"declarations consistent",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Doctor: expected %q in output, got:\n%s", want, output)
		}
	}
}
