//nolint:dupl // similar test structure is intentional
package boilerplate_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestB023_DetectsMissingMiddleware(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	disp := command.NewDispatcher()
	_ = disp
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB023Detector(ctx))
	ruletest.AssertRule(t, findings, "B023", 1)
}

func TestB023_NoFindingWithMiddleware(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	disp := command.NewDispatcher()
	disp.Use(middleware.CommandRecovery())
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB023Detector(ctx))
	ruletest.AssertRule(t, findings, "B023", 0)
}

func TestB024_DetectsMissingBusRecovery(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus := event.NewMemoryBus()
	_ = bus
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB024Detector(ctx))
	ruletest.AssertRule(t, findings, "B024", 1)
}

func TestB024_NoFindingWithRecovery(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	bus := event.NewMemoryBus()
	bus.Use(middleware.EventRecovery())
}
`,
	})
	findings := ruletest.RunDetector(t, boilerplate.NewB024Detector(ctx))
	ruletest.AssertRule(t, findings, "B024", 0)
}
