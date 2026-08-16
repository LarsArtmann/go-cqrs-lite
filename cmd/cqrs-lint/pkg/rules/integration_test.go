package rules_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

// TestIntegration_Taskmanager runs all rules against the example/taskmanager project
// and verifies that findings are produced and the pipeline doesn't crash.
func TestIntegration_Taskmanager(t *testing.T) {
	t.Parallel()

	projectRoot := findProjectRoot(t)
	tmPath := filepath.Join(projectRoot, "example", "taskmanager")

	actx, err := analyzer.BuildContext(tmPath)
	if err != nil {
		t.Fatalf("BuildContext(%s): %v", tmPath, err)
	}

	if len(actx.GoFiles) == 0 {
		t.Skip("no Go files found in taskmanager example (may not be in workspace)")
	}

	detectors := rules.RegisterAll(actx)
	if len(detectors) == 0 {
		t.Fatal("expected at least one detector registered")
	}

	totalFindings := 0
	for _, det := range detectors {
		findings, err := det.Detect(t.Context())
		if err != nil {
			t.Errorf("detector %s: %v", det.Name(), err)

			continue
		}
		totalFindings += len(findings)
	}

	if totalFindings == 0 {
		t.Log(
			"no findings — taskmanager may be very clean, or detectors may not match its patterns",
		)
	}

	t.Logf("ran %d detectors, found %d total findings", len(detectors), totalFindings)
}

// taskmanagerGoldenProfile is the expected finding profile for example/taskmanager.
// It pins the rule→count map so that any change in finding behavior (new FP,
// lost detection, count drift) is caught as a test failure.
// Update with CQRS_LINT_UPDATE_GOLDEN=1 go test -run TestIntegration_TaskmanagerExpectedFindings ./pkg/rules/.
var taskmanagerGoldenProfile = map[string]int{
	"A009": 1,
	"A032": 3,
	"B004": 1,
	"B005": 1,
	"B028": 1,
	"C004": 1,
	"C009": 2,
	"C013": 1,
	"C017": 1,
	"C023": 3,
	"C026": 2,
	"D013": 1,
	"E003": 1,
	"E017": 1,
	"S010": 1,
	"V003": 1,
	"V006": 1,
}

// TestIntegration_TaskmanagerExpectedFindings runs all rules against example/taskmanager
// and verifies the end-to-end finding profile matches a golden snapshot.
// Catches regressions: new false positives, lost detections, and count drift.
// No critical-severity findings or detector errors are allowed.
func TestIntegration_TaskmanagerExpectedFindings(t *testing.T) {
	t.Parallel()

	projectRoot := findProjectRoot(t)
	tmPath := filepath.Join(projectRoot, "example", "taskmanager")

	actx, err := analyzer.BuildContext(tmPath)
	if err != nil {
		t.Fatalf("BuildContext(%s): %v", tmPath, err)
	}

	if len(actx.GoFiles) == 0 {
		t.Skip("no Go files found in taskmanager example")
	}

	detectors := rules.RegisterAll(actx)

	byRule := make(map[string]int)

	for _, det := range detectors {
		findings, detErr := det.Detect(t.Context())
		if detErr != nil {
			t.Errorf("detector %s returned error: %v", det.Name(), detErr)
			continue
		}

		for _, f := range findings {
			byRule[string(f.Rule)]++

			if f.Severity == finding.SeverityCritical {
				t.Errorf("unexpected Critical finding from %s: %s", f.Rule, f.Message)
			}
		}
	}

	if os.Getenv("CQRS_LINT_UPDATE_GOLDEN") == "1" {
		ruleIDs := make([]string, 0, len(byRule))
		for r := range byRule {
			ruleIDs = append(ruleIDs, r)
		}
		sort.Strings(ruleIDs)

		t.Log("paste this into taskmanagerGoldenProfile:")
		for _, r := range ruleIDs {
			t.Logf("\t%q: %d,", r, byRule[r])
		}

		return
	}

	// Check for rules that appeared or disappeared.
	for rule, count := range byRule {
		expected, ok := taskmanagerGoldenProfile[rule]
		if !ok {
			t.Errorf("NEW finding %q fired %d time(s) — not in golden profile. "+
				"If this is an intentional change, update with CQRS_LINT_UPDATE_GOLDEN=1", rule, count)
			continue
		}
		if count != expected {
			t.Errorf("finding count drift for %q: got %d, golden expects %d — "+
				"update with CQRS_LINT_UPDATE_GOLDEN=1", rule, count, expected)
		}
	}

	for rule, expected := range taskmanagerGoldenProfile {
		if _, ok := byRule[rule]; !ok {
			t.Errorf("LOST finding %q was expected %d time(s) but did not fire — "+
				"a rule may have regressed", rule, expected)
		}
	}

	total := 0
	for _, count := range byRule {
		total += count
	}

	t.Logf("total: %d findings across %d rules", total, len(byRule))
}

// TestIntegration_RegisterCritical verifies --fast mode detectors are a subset of all detectors.
func TestIntegration_RegisterCritical(t *testing.T) {
	t.Parallel()

	projectRoot := findProjectRoot(t)
	tmPath := filepath.Join(projectRoot, "example", "taskmanager")

	actx, err := analyzer.BuildContext(tmPath)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if len(actx.GoFiles) == 0 {
		t.Skip("no Go files found")
	}

	all := rules.RegisterAll(actx)
	critical := rules.RegisterCritical(actx)

	if len(critical) >= len(all) {
		t.Errorf(
			"critical detectors (%d) should be fewer than all detectors (%d)",
			len(critical),
			len(all),
		)
	}
}

// TestIntegration_FilterByCategory verifies category filtering works.
func TestIntegration_FilterByCategory(t *testing.T) {
	t.Parallel()

	projectRoot := findProjectRoot(t)
	tmPath := filepath.Join(projectRoot, "example", "taskmanager")

	actx, err := analyzer.BuildContext(tmPath)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if len(actx.GoFiles) == 0 {
		t.Skip("no Go files found")
	}

	all := rules.RegisterAll(actx)
	filtered := rules.FilterByCategory(all, []string{"correctness"})

	if len(filtered) == 0 {
		t.Fatal("expected at least one correctness detector")
	}

	for _, d := range filtered {
		cat := rules.AllRules()
		found := false
		for _, r := range cat {
			if r.ID == d.Name()[:len(r.ID)] || (len(d.Name()) >= 4 && d.Name()[:4] == r.ID) {
				if r.Category == "correctness" {
					found = true

					break
				}
			}
		}
		if !found {
			t.Errorf("detector %s should be in correctness category", d.Name())
		}
	}
}

// TestIntegration_FilterByRuleIDs verifies individual rule ID filtering works.
func TestIntegration_FilterByRuleIDs(t *testing.T) {
	t.Parallel()

	projectRoot := findProjectRoot(t)
	tmPath := filepath.Join(projectRoot, "example", "taskmanager")

	actx, err := analyzer.BuildContext(tmPath)
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if len(actx.GoFiles) == 0 {
		t.Skip("no Go files found")
	}

	all := rules.RegisterAll(actx)
	filtered := rules.FilterByRuleIDs(all, []string{"C001", "C002"})

	if len(filtered) != 2 {
		t.Fatalf("expected 2 detectors for C001,C002, got %d", len(filtered))
	}

	if !strings.HasPrefix(filtered[0].Name(), "C001") {
		t.Errorf("first detector should be C001, got %s", filtered[0].Name())
	}
}

// TestUnit_FilterByRuleIDs verifies individual rule ID filtering works without needing a real project.
func TestUnit_FilterByRuleIDs(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	all := rules.RegisterAll(ctx)
	if len(all) == 0 {
		t.Fatal("expected detectors from RegisterAll")
	}

	filtered := rules.FilterByRuleIDs(all, []string{"C001", "C002"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 detectors for C001,C002, got %d", len(filtered))
	}

	for _, d := range filtered {
		if !strings.HasPrefix(d.Name(), "C00") {
			t.Errorf("detector %s should start with C00", d.Name())
		}
	}
}

// TestUnit_IsRuleID tests the rule ID detection function.
func TestUnit_IsRuleID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"C001", true},
		{"A019", true},
		{"S003", true},
		{"E007", true},
		{"B015", true},
		{"D005", true},
		{"correctness", false},
		{"api", false},
		{"C01", false},
		{"c001", true},
		{"", false},
	}
	for _, tt := range tests {
		if got := rules.IsRuleID(tt.input); got != tt.want {
			t.Errorf("IsRuleID(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}

	return root
}
