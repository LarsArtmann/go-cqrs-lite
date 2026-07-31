package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC034_DetectsGoroutineWithoutCtx(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, cmd *Command) error {
	go func() {
		process(cmd)
	}()
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC034Detector(ctx))
	ruletest.AssertRule(t, findings, "C034", 1)
}

func TestC034_NoFindingWhenCtxPassed(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, cmd *Command) error {
	go process(ctx, cmd)
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC034Detector(ctx))
	ruletest.AssertRule(t, findings, "C034", 0)
}

func TestC034_NoFindingForFuncWithoutCtxParam(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func setup() {
	go func() {
		println("background")
	}()
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC034Detector(ctx))
	ruletest.AssertRule(t, findings, "C034", 0)
}

func TestC034_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC034Detector(ctx))
	ruletest.AssertRule(t, findings, "C034", 0)
}
