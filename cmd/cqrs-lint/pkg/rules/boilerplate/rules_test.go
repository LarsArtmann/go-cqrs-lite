package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestB001_DetectsSingleEventHelper(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func singleEvent(type_ string, id string, streamType string, ver event.Version, payload any) []event.Event {
	evt, _ := event.New(type_, id, streamType, ver, payload)
	return []event.Event{evt}
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB001Detector(ctx))
	ruletest.AssertRule(t, findings, "B001", 1)
}
