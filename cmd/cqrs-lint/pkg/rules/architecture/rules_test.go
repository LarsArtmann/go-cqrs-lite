package architecture_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/architecture"
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

func TestE004_DetectsUncatalogedEvent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func createEvent() {
	event.New("user.created", "id", "User", event.Version(1), struct{}{})
}
`,
	})
	findings := runDetector(t, architecture.NewE004Detector(ctx))
	assertRule(t, findings, "E004", 1)
}
