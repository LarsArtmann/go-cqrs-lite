package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestP006_DetectsShortPollingInterval(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"poller.go": `package main

import "time"

func poll() {
	for {
		if ready() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP006Detector(ctx))
	ruletest.AssertRule(t, findings, "P006", 1)
}

func TestP006_NoFindingForLongInterval(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"poller.go": `package main

import "time"

func poll() {
	for {
		if ready() {
			break
		}
		time.Sleep(5 * time.Second)
	}
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP006Detector(ctx))
	ruletest.AssertRule(t, findings, "P006", 0)
}

func TestP006_NoFindingForSleepOutsideLoop(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"util.go": `package main

import "time"

func wait() {
	time.Sleep(10 * time.Millisecond)
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP006Detector(ctx))
	ruletest.AssertRule(t, findings, "P006", 0)
}
