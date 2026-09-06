// Package ruletest provides shared test helpers for cqrs-lint rule tests.
// All rule subpackages (correctness, security, architecture, etc.) use
// RunDetector and AssertRule to eliminate per-package boilerplate.
package ruletest

import (
	"context"
	"fmt"
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

// AliasedImportSource renders a single-file fixture body with pkgPath
// imported under the given alias. Several detectors must resolve
// qualifiers through the import table to survive aliasing (the A014
// alias-blindness bug class); this helper keeps those fixtures to one
// line instead of a hand-written import block per test.
func AliasedImportSource(alias, pkgPath, body string) string {
	return fmt.Sprintf("package main\n\nimport %s %q\n\n%s\n", alias, pkgPath, body)
}
