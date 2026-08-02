package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC039_GoroutineInHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func SubscribeAll(evt event.Event) {
	go func() {
		processAsync(evt)
	}()
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC039Detector(ctx))
	ruletest.AssertRule(t, findings, "C039", 1)
}

func TestC039_GoroutineWithWaitGroupNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func SubscribeAll(evt event.Event) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		processAsync(evt)
	}()
	wg.Wait()
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC039Detector(ctx))
	ruletest.AssertRule(t, findings, "C039", 0)
}

func TestC039_GoroutineWithCtxDoneNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func SubscribeAll(ctx context.Context, evt event.Event) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case ch <- evt:
		}
	}()
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC039Detector(ctx))
	ruletest.AssertRule(t, findings, "C039", 0)
}

func TestC039_NoGoroutineNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func SubscribeAll(evt event.Event) {
	process(evt)
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC039Detector(ctx))
	ruletest.AssertRule(t, findings, "C039", 0)
}

func TestC039_GoroutineInNonHandlerNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func runWorker() {
	go func() {
		work()
	}()
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC039Detector(ctx))
	ruletest.AssertRule(t, findings, "C039", 0)
}
