package version_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"
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
	}
}
