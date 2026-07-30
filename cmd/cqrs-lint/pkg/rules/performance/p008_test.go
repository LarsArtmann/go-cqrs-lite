package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
)

func TestP008_DetectsMissingBatchSize(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	host, _ := projectionhost.New(journal, cpStore)
	_ = host
}
`,
	})
	findings := runDetector(t, performance.NewP008Detector(ctx))
	assertRule(t, findings, "P008", 1)
}

func TestP008_NoFindingWhenBatchSizeSet(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	host, _ := projectionhost.New(journal, cpStore,
		projectionhost.WithBatchSize(500))
	_ = host
}
`,
	})
	findings := runDetector(t, performance.NewP008Detector(ctx))
	assertRule(t, findings, "P008", 0)
}

func TestP008_NoFindingForOtherNewCalls(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store := somestore.New()
	_ = store
}
`,
	})
	findings := runDetector(t, performance.NewP008Detector(ctx))
	assertRule(t, findings, "P008", 0)
}
