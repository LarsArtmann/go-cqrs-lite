package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC032_DetectsContextBackgroundInHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, cmd *Command) error {
	bgCtx := context.Background()
	return process(bgCtx, cmd)
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC032Detector(ctx))
	ruletest.AssertRule(t, findings, "C032", 1)
}

func TestC032_DetectsContextTODOInHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, cmd *Command) error {
	todoCtx := context.TODO()
	return process(todoCtx, cmd)
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC032Detector(ctx))
	ruletest.AssertRule(t, findings, "C032", 1)
}

func TestC032_NoFindingWhenCtxPropagated(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, cmd *Command) error {
	return process(ctx, cmd)
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC032Detector(ctx))
	ruletest.AssertRule(t, findings, "C032", 0)
}

func TestC032_NoFindingForFunctionWithoutContextParam(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "context"

func setup() {
	bgCtx := context.Background()
	_ = bgCtx
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC032Detector(ctx))
	ruletest.AssertRule(t, findings, "C032", 0)
}

func TestC032_NoFindingForNonHandlerWithContextParam(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "context"

func backgroundWorker(ctx context.Context) {
	detached := context.Background()
	go run(detached)
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC032Detector(ctx))
	ruletest.AssertRule(t, findings, "C032", 0)
}

func TestC032_DetectsInProjectionMethod(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "context"

type UserProjection struct{}

func (p *UserProjection) Sync(ctx context.Context, evt Event) error {
	bg := context.Background()
	return apply(bg, evt)
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC032Detector(ctx))
	ruletest.AssertRule(t, findings, "C032", 1)
}

func TestC032_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC032Detector(ctx))
	ruletest.AssertRule(t, findings, "C032", 0)
}
