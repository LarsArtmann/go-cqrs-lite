package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
)

func TestP007_DetectsBitshiftBackoff(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"storage.go": `package main

import "time"

func appendWithRetry(baseBackoff time.Duration, data []byte) {
	for attempt := 0; attempt < 3; attempt++ {
		delay := baseBackoff << time.Duration(attempt)
		time.Sleep(delay)
	}
}
`,
	})
	findings := runDetector(t, performance.NewP007Detector(ctx))
	assertRule(t, findings, "P007", 1)
}

func TestP007_NoFindingForNormalBitshift(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"util.go": `package main

func bitmask(n int) int {
	return 1 << n
}
`,
	})
	findings := runDetector(t, performance.NewP007Detector(ctx))
	assertRule(t, findings, "P007", 0)
}
