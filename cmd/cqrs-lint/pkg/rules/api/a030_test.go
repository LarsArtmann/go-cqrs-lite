package api_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestA030_DetectsStrategyWithoutStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, d,
		decider.WithSnapshotStrategy(strategy))
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA030Detector(ctx))
	ruletest.AssertRule(t, findings, "A030", 1)
}

func TestA030_NoFindingWhenStoreAndStrategyPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, d,
		decider.WithSnapshotStore(snapStore),
		decider.WithSnapshotStrategy(strategy))
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA030Detector(ctx))
	ruletest.AssertRule(t, findings, "A030", 0)
}

func TestA030_NoFindingWhenNoStrategy(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	repo, _ := decider.NewRepository(store, bus, d,
		decider.WithStateCache(cache))
	_ = repo
}
`,
	})
	findings := ruletest.RunDetector(t, api.NewA030Detector(ctx))
	ruletest.AssertRule(t, findings, "A030", 0)
}

func TestA030_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, api.NewA030Detector(ctx))
	ruletest.AssertRule(t, findings, "A030", 0)
}
