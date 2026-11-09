package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
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

// TestD011_DetectsNilPayloadViaNew pins the event.New half of the surface
// (the previous test only exercised NewEvent).
func TestD011_DetectsNilPayloadViaNew(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func emit() {
	_, _ = event.New("user.toggled", "id1", "User", 1, nil)
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD011Detector(ctx))
	ruletest.AssertRule(t, findings, "D011", 1)
}

// TestD011_AliasedImportStillFires pins alias resilience: the detector must
// resolve the qualifier through the import table, not compare the literal
// name "event" (the A014 alias-blindness bug class).
func TestD011_AliasedImportStillFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": ruletest.AliasedImportSource(
			"cqrs",
			"github.com/larsartmann/go-cqrs-lite/event/v4",
			`func emit() {
	_, _ = cqrs.NewEvent("user.toggled", "id1", "User", 1, nil)
}`,
		),
	})
	findings := ruletest.RunDetector(t, consistency.NewD011Detector(ctx))
	ruletest.AssertRule(t, findings, "D011", 1)
}

// TestD011_ForeignEventPackageStaysSilent: a call on an UNRELATED package
// that happens to be named event must not fire.
func TestD011_ForeignEventPackageStaysSilent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "example.com/otherepo/project/event"

func emit() {
	_ = event.NewEvent("user.toggled", "id1", "User", 1, nil)
}
`,
	})
	findings := ruletest.RunDetector(t, consistency.NewD011Detector(ctx))
	ruletest.AssertRule(t, findings, "D011", 0)
}
