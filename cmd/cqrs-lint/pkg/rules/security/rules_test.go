package security_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
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

func TestS001_DetectsHardcodedSecret(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

func init() {
	apiKey := "super-secret-key-1234567890"
	_ = apiKey
}
`,
	})
	findings := runDetector(t, security.NewS001Detector(ctx))
	assertRule(t, findings, "S001", 1)
}

func TestS001_NoFindingForShortString(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

func init() {
	token := "abc"
	_ = token
}
`,
	})
	findings := runDetector(t, security.NewS001Detector(ctx))
	assertRule(t, findings, "S001", 0)
}
