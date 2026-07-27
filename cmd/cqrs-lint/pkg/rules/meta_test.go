package rules

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestAllDetectorsInstantiate verifies that every detector constructor
// returns a non-nil detector with a non-empty name. This is a compile-time
// and runtime guard against broken detector constructors (e.g., the b3931503
// incident where slices.Contains() was called with zero arguments).
func TestAllDetectorsInstantiate(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{}
	detectors := RegisterAll(ctx)

	if len(detectors) != 65 {
		t.Fatalf("expected 65 detectors, got %d", len(detectors))
	}

	for _, d := range detectors {
		if d == nil {
			t.Fatal("detector is nil")
		}

		name := d.Name()
		if name == "" {
			t.Fatal("detector has empty name")
		}

		// Every detector name must start with a rule ID prefix (e.g., "C001-")
		if len(name) < 5 {
			t.Fatalf("detector name too short: %q", name)
		}

		// Must start with uppercase letter + 3 digits + dash
		if name[0] < 'A' || name[0] > 'Z' {
			t.Fatalf("detector name must start with uppercase letter: %q", name)
		}

		for i := 1; i < 4; i++ {
			if name[i] < '0' || name[i] > '9' {
				t.Fatalf("detector name must have digits at positions 1-3: %q", name)
			}
		}
	}
}

// TestCriticalDetectorsInstantiate verifies the --fast mode subset.
func TestCriticalDetectorsInstantiate(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{}
	detectors := RegisterCritical(ctx)

	if len(detectors) != 5 {
		t.Fatalf("expected 5 critical detectors, got %d", len(detectors))
	}

	for _, d := range detectors {
		if d == nil {
			t.Fatal("critical detector is nil")
		}

		if d.Name() == "" {
			t.Fatal("critical detector has empty name")
		}
	}
}

// TestCatalogCountMatchesRegister verifies that the catalog (user-facing rule
// metadata) and RegisterAll (actual detectors) agree on the rule count. This
// catches drift when a rule is added to the catalog but not registered (or
// vice versa).
func TestCatalogCountMatchesRegister(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{}

	catalogRules := AllRules()
	detectors := RegisterAll(ctx)

	if len(catalogRules) != len(detectors) {
		t.Fatalf("catalog has %d rules but RegisterAll returned %d detectors "+
			"(catalog IDs: %v)",
			len(catalogRules), len(detectors), catalogRuleIDs(catalogRules))
	}

	catalogIDs := make(map[string]bool, len(catalogRules))
	for _, r := range catalogRules {
		if catalogIDs[r.ID] {
			t.Fatalf("duplicate catalog rule ID: %s", r.ID)
		}
		catalogIDs[r.ID] = true
	}

	for _, d := range detectors {
		name := d.Name()
		if len(name) < 4 {
			continue
		}

		ruleID := name[:4]
		if !catalogIDs[ruleID] {
			t.Fatalf("detector %q has rule ID %s not in catalog", name, ruleID)
		}
	}
}

func catalogRuleIDs(rules []RuleInfo) []string {
	ids := make([]string, len(rules))
	for i, r := range rules {
		ids[i] = r.ID
	}

	return ids
}
