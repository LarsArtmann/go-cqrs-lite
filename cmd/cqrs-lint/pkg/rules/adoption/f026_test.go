package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestF026_NewReaderWithoutPrefetchFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.NewReader[any](nil, "items")
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, adoption.NewF026Detector(ctx))
	ruletest.AssertRule(t, findings, "F026", 1)
}

func TestF026_NoFindingWithPrefetch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	r := metaengine.NewReader[any](nil, "items")
	_ = r
	_ = metaengine.WithPrefetch(100)
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, adoption.NewF026Detector(ctx))
	ruletest.AssertRule(t, findings, "F026", 0)
}

func TestF026_NoFindingWithoutMetaengine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF026Detector(ctx))
	ruletest.AssertRule(t, findings, "F026", 0)
}
