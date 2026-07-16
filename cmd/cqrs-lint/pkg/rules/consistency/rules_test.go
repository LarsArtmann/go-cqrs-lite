package consistency_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
)

func runDetector(t *testing.T, det finding.Detector) []finding.Finding {
	t.Helper()
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("detector %s: %v", det.Name(), err)
	}

	return findings
}

func assertRule(t *testing.T, findings []finding.Finding, ruleID string, wantCount int) {
	t.Helper()
	count := 0
	for _, f := range findings {
		if string(f.Rule) == ruleID {
			count++
		}
	}
	if count != wantCount {
		t.Errorf("rule %s: got %d findings, want %d", ruleID, count, wantCount)
		for _, f := range findings {
			t.Logf("  finding: %s %s: %s", f.Rule, f.Severity, f.Message)
		}
	}
}

func TestD002_DetectsMixedJSONCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type User struct {
	FirstName string ` + "`json:\"first_name\"`" + `
	LastName  string ` + "`json:\"lastName\"`" + `
}
`,
	})
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 1)
}

func TestD002_NoFindingForConsistentCasing(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type User struct {
	FirstName string ` + "`json:\"firstName\"`" + `
	LastName  string ` + "`json:\"lastName\"`" + `
}
`,
	})
	findings := runDetector(t, consistency.NewD002Detector(ctx))
	assertRule(t, findings, "D002", 0)
}
