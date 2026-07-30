package rules

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"

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

	if len(detectors) != 100 {
		t.Fatalf("expected 100 detectors, got %d", len(detectors))
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
// metadata) and RegisterAll (actual detectors) agree on every rule ID
// bidirectionally. This catches drift when a rule is added to the catalog but
// not registered (or vice versa).
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

	// Build catalog ID set, checking for duplicates.
	catalogIDs := make(map[string]bool, len(catalogRules))
	for _, r := range catalogRules {
		if catalogIDs[r.ID] {
			t.Fatalf("duplicate catalog rule ID: %s", r.ID)
		}
		catalogIDs[r.ID] = true
	}

	// Build detector ID set, checking for duplicates.
	detectorIDs := make(map[string]bool, len(detectors))
	for _, d := range detectors {
		name := d.Name()
		if len(name) < 4 {
			t.Fatalf("detector name too short to extract rule ID: %q", name)
		}

		ruleID := name[:4]

		if detectorIDs[ruleID] {
			t.Fatalf("duplicate detector rule ID: %s", ruleID)
		}
		detectorIDs[ruleID] = true
	}

	// Forward: every detector's rule ID must be in the catalog.
	for id := range detectorIDs {
		if !catalogIDs[id] {
			t.Fatalf("detector rule ID %s is not in the catalog", id)
		}
	}

	// Reverse: every catalog rule ID must have a registered detector.
	for id := range catalogIDs {
		if !detectorIDs[id] {
			t.Fatalf("catalog rule ID %s has no registered detector", id)
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

// TestCatalogSeverityAndConfidenceValid verifies that every catalog entry's
// Severity and Confidence strings map to valid finding.Severity and
// finding.Confidence values. Catches typos like "warn" instead of "warning"
// or "hi" instead of "high" that would silently break filtering and the
// health score computation.
func TestCatalogSeverityAndConfidenceValid(t *testing.T) {
	t.Parallel()

	validSeverities := map[string]finding.Severity{
		"critical": finding.SeverityCritical,
		"error":    finding.SeverityError,
		"warning":  finding.SeverityWarning,
		"info":     finding.SeverityInfo,
	}

	validConfidences := map[string]finding.Confidence{
		"high":   finding.ConfidenceHigh,
		"medium": finding.ConfidenceMedium,
		"low":    finding.ConfidenceLow,
	}

	for _, r := range AllRules() {
		if _, ok := validSeverities[strings.ToLower(r.Severity)]; !ok {
			t.Errorf(
				"rule %s: invalid severity %q (must be critical/error/warning/info)",
				r.ID,
				r.Severity,
			)
		}

		if _, ok := validConfidences[strings.ToLower(r.Confidence)]; !ok {
			t.Errorf("rule %s: invalid confidence %q (must be high/medium/low)", r.ID, r.Confidence)
		}

		if r.Category == "" {
			t.Errorf("rule %s: empty category", r.ID)
		}

		if r.Description == "" {
			t.Errorf("rule %s: empty description", r.ID)
		}

		if r.Name == "" {
			t.Errorf("rule %s: empty name", r.ID)
		}
	}
}

// TestCriticalDetectorsAreCriticalOrError verifies that every detector in
// RegisterCritical maps to a catalog entry with critical or error severity.
// The --fast mode promises "critical correctness rules only"; including a
// warning or info rule would be misleading.
func TestCriticalDetectorsAreCriticalOrError(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{}
	detectors := RegisterCritical(ctx)

	catalogByID := make(map[string]RuleInfo, len(AllRules()))
	for _, r := range AllRules() {
		catalogByID[r.ID] = r
	}

	for _, d := range detectors {
		name := d.Name()
		if len(name) < 4 {
			t.Fatalf("detector name too short: %q", name)
		}

		ruleID := name[:4]
		info, ok := catalogByID[ruleID]
		if !ok {
			t.Fatalf("critical detector %s not in catalog", ruleID)
		}

		sev := strings.ToLower(info.Severity)
		if sev != "critical" && sev != "error" {
			t.Errorf(
				"critical detector %s has severity %q in catalog — expected critical or error",
				ruleID,
				info.Severity,
			)
		}
	}
}

// TestDetectorNamesMatchCatalog verifies that each detector's name suffix
// matches the catalog Name field. The detector name format is "C001-name"
// and the catalog Name is "name". A mismatch means the detector and catalog
// describe different rules.
func TestDetectorNamesMatchCatalog(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{}
	detectors := RegisterAll(ctx)

	catalogByID := make(map[string]RuleInfo, len(AllRules()))
	for _, r := range AllRules() {
		catalogByID[r.ID] = r
	}

	for _, d := range detectors {
		name := d.Name()
		if len(name) < 5 || name[4] != '-' {
			t.Fatalf("detector name %q does not follow ID-name format", name)
		}

		ruleID := name[:4]
		detectorSuffix := name[5:]

		info, ok := catalogByID[ruleID]
		if !ok {
			t.Fatalf("detector %s not in catalog", ruleID)
		}

		if detectorSuffix != info.Name {
			t.Errorf(
				"detector %s: name suffix %q != catalog name %q",
				ruleID,
				detectorSuffix,
				info.Name,
			)
		}
	}
}
