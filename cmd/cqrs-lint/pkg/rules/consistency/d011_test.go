package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
)

func TestD011_DetectsNilPayload(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func emit() {
	_ = event.NewEvent("user.toggled", "id1", "User", 1, nil)
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD011Detector(ctx))
	ruletest.AssertRule(t, findings, "D011", 1)
}

func TestD011_NoFindingForTypedPayload(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type TogglePayload struct{ Active bool }

func emit() {
	_ = event.New("user.toggled", "id1", "User", 1, TogglePayload{Active: true})
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD011Detector(ctx))
	ruletest.AssertRule(t, findings, "D011", 0)
}
