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
	findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
	ruletest.AssertRule(t, findings, "C030", 1)
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
	findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
	ruletest.AssertRule(t, findings, "C030", 0)
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
	findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
	ruletest.AssertRule(t, findings, "C030", 0)
}

func TestC030_NoFindingWhenDoneOnNonCtxReceiver(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"worker.go": `package main

import "net/http"

func handler(w http.ResponseWriter, r *http.Request) {
	for {
		select {
		case <-r.Context().Done():
			return
		}
	}
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
	ruletest.AssertRule(t, findings, "C030", 0)
}

func TestC030_NoFindingWhenCtxErrCheck(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"worker.go": `package main

import "context"

func poll(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		doWork()
	}
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
	ruletest.AssertRule(t, findings, "C030", 0)
}

func TestC030_NoFindingWhenLoopHasBreak(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"worker.go": `package main

func reconstruct(parent map[int]int, end int) []int {
	var path []int
	for k := end; ; k = parent[k] {
		path = append(path, k)
		if k == parent[k] {
			break
		}
	}
	return path
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
	ruletest.AssertRule(t, findings, "C030", 0)
}

func TestC030_NoFindingWhenCustomStopChannel(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"worker.go": `package main

import "time"

func sampler(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			doWork()
		}
	}
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
	ruletest.AssertRule(t, findings, "C030", 0)
}

func TestC030_StillFlagsLoopWithOnlyReturnInGoroutine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"worker.go": `package main

func worker() {
	for {
		go func() {
			return
		}()
		doWork()
	}
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC030Detector(ctx))
	ruletest.AssertRule(t, findings, "C030", 1)
}
