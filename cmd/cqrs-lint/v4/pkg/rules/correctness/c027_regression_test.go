package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestC027_NoFindingForNonEventBus(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"main.go": `package main

type projectionhostPkg struct{}

func (projectionhostPkg) New(args ...any) (any, error) { return nil, nil }

type ErrorBus struct{}

func (*ErrorBus) Subscribe(eventType string, fn func(string)) {}

func setup() {
	projectionhost := projectionhostPkg{}
	_, _ = projectionhost.New(nil, nil)

	errBus := &ErrorBus{}
	errBus.Subscribe("user.created", func(s string) {})
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, correctness.NewC027Detector(ctx))
	ruletest.AssertRule(t, findings, "C027", 0)
}

func TestC027_DetectsEventBusSubscribeWithProjectionHost(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func setup() {
	projectionhost.New(journal, cpStore)
	bus.Subscribe("user.created", handler)
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC027Detector(ctx))
	ruletest.AssertRule(t, findings, "C027", 1)
}
