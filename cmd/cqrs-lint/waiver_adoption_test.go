package main

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
)

func TestFilterWaivedAdoption_SuppressesWaivedModuleFindings(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		mustFinding("F003", "no otel tracing"),
		mustFinding("F019", "planner tuning"),
	}

	active, waived := filterWaivedAdoption(findings, []analyzer.ScorecardWaiver{
		{
			Key:     "otel",
			Reason:  "single-process CLI, no tracing budget",
			Trigger: "second service lands",
		},
	})

	if len(active) != 1 || string(active[0].Rule) != "F019" {
		t.Fatalf("non-waived adoption findings must keep firing, active = %+v", active)
	}

	if len(waived) != 1 || string(waived[0].Rule) != "F003" {
		t.Fatalf("waived module's finding must be suppressed, waived = %+v", waived)
	}

	suppression := waived[0].Suppression
	if suppression == nil {
		t.Fatal("waived finding must carry a Suppression record")
	}

	if !strings.Contains(suppression.Reason, "single-process CLI, no tracing budget") {
		t.Errorf("suppression reason must carry the waiver reason, got %q", suppression.Reason)
	}

	if !strings.Contains(suppression.Reason, "revisit when: second service lands") {
		t.Errorf("suppression reason must carry the revisit trigger, got %q", suppression.Reason)
	}
}

func TestFilterWaivedAdoption_NoWaiversPassthrough(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{mustFinding("F003", "no otel tracing")}

	active, waived := filterWaivedAdoption(findings, nil)
	if len(active) != 1 || len(waived) != 0 {
		t.Fatalf("no waivers must pass through, active=%d waived=%d", len(active), len(waived))
	}
}

func TestFilterWaivedAdoption_WaiverWithoutAdoptionRuleIsInert(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{mustFinding("F003", "no otel tracing")}

	// "watermill" has no F-series mapping today: the waiver only affects the
	// scorecard render, findings keep firing.
	active, waived := filterWaivedAdoption(findings, []analyzer.ScorecardWaiver{
		{Key: "watermill", Reason: "no broker in this deployment"},
	})

	if len(active) != 1 || len(waived) != 0 {
		t.Fatalf("unmapped waiver key must be inert, active=%d waived=%d", len(active), len(waived))
	}
}

func TestFilterWaivedAdoption_AllSevenMappings(t *testing.T) {
	t.Parallel()

	expected := map[string]string{
		"otel":        "F003",
		"prometheus":  "F004",
		"schema":      "F005",
		"encryption":  "F006",
		"idempotency": "F007",
		"scheduling":  "F009",
		"listing":     "F016",
	}

	for key, rule := range expected {
		if mapped, ok := waiverAdoptionRules[key]; !ok || mapped != rule {
			t.Errorf("waiverAdoptionRules[%q] = %q, want %q", key, mapped, rule)
		}
	}

	if len(waiverAdoptionRules) != len(expected) {
		t.Errorf("mapping grew without a test: %d entries, test pins %d",
			len(waiverAdoptionRules), len(expected))
	}
}

// TestWaiverAdoptionRulesHandMapPinned pins the hand-map POLICY (M24 source
// review): adoption catalog entries carry no machine-readable module key
// (their module link lives in prose), so waiverAdoptionRules cannot be
// GENERATED from the catalog. This companion to AllSevenMappings kills the
// drift classes a generator would: every waiver key must be a real
// module-catalog key, every rule value must be a real adoption rule ID, and
// no rule may be mapped twice (one waiver must not silence two modules'
// findings).
func TestWaiverAdoptionRulesHandMapPinned(t *testing.T) {
	t.Parallel()

	moduleKeys := make(map[string]bool)
	for _, entry := range analyzer.DefaultCatalog.All() {
		moduleKeys[string(entry.Key)] = true
	}

	adoptionRuleIDs := make(map[string]bool)
	for _, rule := range rules.AllRules() {
		if rule.Category == "adoption" {
			adoptionRuleIDs[rule.ID] = true
		}
	}

	seenRules := make(map[string]string, len(waiverAdoptionRules))

	for key, rule := range waiverAdoptionRules {
		if !moduleKeys[key] {
			t.Errorf("waiver key %q is not a module-catalog key", key)
		}

		if !adoptionRuleIDs[rule] {
			t.Errorf("waiver rule %q (key %q) is not an adoption rule ID", rule, key)
		}

		if other, dup := seenRules[rule]; dup {
			t.Errorf("rule %q mapped from both %q and %q — one waiver must not silence two modules' findings", rule, other, key)
		}

		seenRules[rule] = key
	}
}
