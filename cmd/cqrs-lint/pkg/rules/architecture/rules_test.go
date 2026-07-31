package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestE004_DetectsUncatalogedEvent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func createEvent() {
	event.New("user.created", "id", "User", event.Version(1), struct{}{})
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE004Detector(ctx))
	ruletest.AssertRule(t, findings, "E004", 1)
}
