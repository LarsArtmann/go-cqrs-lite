package api_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestA005_NoFindingForBroadcastOnlySubscriber(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func setup() {
	bus.SubscribeAll(func(evt Event) {
		broadcaster.Broadcast(evt)
	})
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA005Detector(ctx))
	ruletest.AssertRule(t, findings, "A005", 0)
}

func TestA005_DetectsProjectionSubscriber(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func setup() {
	bus.SubscribeAll(func(evt Event) {
		store.Set(evt.ID, evt)
	})
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA005Detector(ctx))
	ruletest.AssertRule(t, findings, "A005", 1)
}

func TestA005_NoFindingForNonEventBus(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"handler.go": `package main

type ErrorBus struct{}

func (*ErrorBus) SubscribeAll(fn func(string)) {}

func setup() {
	errBus := &ErrorBus{}
	errBus.SubscribeAll(func(s string) {
		println(s)
	})
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, api.NewA005Detector(ctx))
	ruletest.AssertRule(t, findings, "A005", 0)
}
