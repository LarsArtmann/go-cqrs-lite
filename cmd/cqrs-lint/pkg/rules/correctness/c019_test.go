package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC019_DetectsDuplicateRepoType(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handlers.go": `package main

func setup1() {
	repo1 := decider.NewRepository[UserState](store1, bus1, d1)
}

func setup2() {
	repo2 := decider.NewRepository[UserState](store2, bus2, d2)
}
`,
	})
	findings := runDetector(t, correctness.NewC019Detector(ctx))
	assertRule(t, findings, "C019", 1)
}

func TestC019_NoFindingForSingleRepo(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handlers.go": `package main

func setup() {
	repo := decider.NewRepository[UserState](store, bus, d)
}
`,
	})
	findings := runDetector(t, correctness.NewC019Detector(ctx))
	assertRule(t, findings, "C019", 0)
}

func TestC019_NoFindingForDifferentTypes(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handlers.go": `package main

func setup() {
	userRepo := decider.NewRepository[UserState](s, b, d1)
	orderRepo := decider.NewRepository[OrderState](s, b, d2)
}
`,
	})
	findings := runDetector(t, correctness.NewC019Detector(ctx))
	assertRule(t, findings, "C019", 0)
}
