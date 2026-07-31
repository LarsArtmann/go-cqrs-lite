package correctness_test //nolint:dupl // test boilerplate

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC029_DetectsNilKeyExtractor(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(store interface{}, ttl int) {
	qry.Use(middleware.QueryIdempotency(store, ttl, nil))
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC029Detector(ctx))
	ruletest.AssertRule(t, findings, "C029", 1)
}

func TestC029_NoFindingWhenKeyExtractorProvided(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(store interface{}, ttl int) {
	qry.Use(middleware.QueryIdempotency(store, ttl, keyFn))
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC029Detector(ctx))
	ruletest.AssertRule(t, findings, "C029", 0)
}

func TestC029_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC029Detector(ctx))
	ruletest.AssertRule(t, findings, "C029", 0)
}
