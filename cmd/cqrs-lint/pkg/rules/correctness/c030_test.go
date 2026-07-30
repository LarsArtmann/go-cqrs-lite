package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC030_DetectsInfiniteLoopWithoutCancel(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"worker.go": `package main

import "context"

func worker(ctx context.Context) {
	for {
		doWork()
	}
}
`,
	})
	findings := runDetector(t, correctness.NewC030Detector(ctx))
	assertRule(t, findings, "C030", 1)
}

func TestC030_NoFindingWhenCtxDoneInSelect(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"worker.go": `package main

import "context"

func worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			doWork()
		}
	}
}
`,
	})
	findings := runDetector(t, correctness.NewC030Detector(ctx))
	assertRule(t, findings, "C030", 0)
}

func TestC030_NoFindingForBoundedLoop(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"worker.go": `package main

func worker() {
	for i := 0; i < 10; i++ {
		doWork()
	}
}
`,
	})
	findings := runDetector(t, correctness.NewC030Detector(ctx))
	assertRule(t, findings, "C030", 0)
}
