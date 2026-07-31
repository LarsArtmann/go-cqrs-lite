package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

// B001 must NOT fire on a function whose name merely CONTAINS a target name as
// a substring. Previously dismissSingleEvent matched "singleEvent" and was
// flagged. Exact case-insensitive match only.
func TestB001_NoFindingForSubstringName(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func dismissSingleEvent(type_ string, id string, streamType string, ver event.Version, payload any) {
	evt, _ := event.New(type_, id, streamType, ver, payload)
	_ = evt
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB001Detector(ctx))
	ruletest.AssertRule(t, findings, "B001", 0)
}

// B011 must NOT fire on a must*-prefixed helper that RETURNS the marshal error
// instead of panicking. The rule's message claims the function "panics on
// marshal error" — without an actual panic() that is a false claim.
func TestB011_NoFindingWhenErrorReturnedNotPanicked(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"helper.go": `package main

import "encoding/json"

func mustMarshal(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB011Detector(ctx))
	ruletest.AssertRule(t, findings, "B011", 0)
}
