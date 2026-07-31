// Package ruletest provides shared test helpers for cqrs-lint rule tests.
// All rule subpackages (correctness, security, architecture, etc.) use
// RunDetector and AssertRule to eliminate per-package boilerplate.
package ruletest

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"
)

// RunDetector executes a [finding.Detector] and fails the test on error.
func RunDetector(t *testing.T, det finding.Detector) []finding.Finding {
	t.Helper()

	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("detector %s: %v", det.Name(), err)
	}

	return findings
}

// AssertRule asserts that exactly wantCount findings match ruleID.
// On mismatch it logs every finding to aid debugging.
func AssertRule(t *testing.T, findings []finding.Finding, ruleID string, wantCount int) {
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
