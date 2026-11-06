package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestF026_NewReaderWithoutPrefetchFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	r := metaengine.NewReader[any](nil, "items")
	_, _, _ = r.ScanPage(context.Background(), metaengine.WithLimit(10))
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, adoption.NewF026Detector(ctx))
	ruletest.AssertRule(t, findings, "F026", 1)
}

func TestF026_NoFindingForPointGetOnlyReader(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	r := metaengine.NewReader[any](nil, "items")
	_, _, _ = r.Get(context.Background(), "k")
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, adoption.NewF026Detector(ctx))
	ruletest.AssertRule(t, findings, "F026", 0)
}

// TestF026_NoFindingWithPrefetch uses the reader-level form: the shape that has
// existed since the cache shipped.
func TestF026_NoFindingWithPrefetch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	r := metaengine.NewReader[any](nil, "items").WithPrefetch(metaengine.NewPrefetchCache())
	_, _ = r.Scan(context.Background(), metaengine.WithLimit(10))
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, adoption.NewF026Detector(ctx))
	ruletest.AssertRule(t, findings, "F026", 0)
}

// TestF026_NoFindingWithPrefetchScanOption covers the per-call form the rule
// coaches (metaengine >= v4.21.0).
func TestF026_NoFindingWithPrefetchScanOption(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	r := metaengine.NewReader[any](nil, "items")
	_, _ = r.Scan(context.Background(),
		metaengine.WithPrefetch(metaengine.NewPrefetchCache()),
		metaengine.WithLimit(10))
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, adoption.NewF026Detector(ctx))
	ruletest.AssertRule(t, findings, "F026", 0)
}

// TestF026_AliasedImportStillDetected pins the alias fix: `metaeng` must not
// smuggle a reader past the rule.
func TestF026_AliasedImportStillDetected(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	metaeng "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	r := metaeng.NewReader[any](nil, "items")
	_, _, _ = r.ScanPage(context.Background(), metaeng.WithLimit(10))
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, adoption.NewF026Detector(ctx))
	ruletest.AssertRule(t, findings, "F026", 1)
}

// TestF026_NoFindingForOneShotScan pins the precision fix: a single filtered
// Scan (no cursor, no next page) gains nothing from a prefetch cache and would
// pay its doubled fetch, so it is not coached (the example/taskmanager shape).
func TestF026_NoFindingForOneShotScan(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	r := metaengine.NewReader[any](nil, "items")
	_, _ = r.Scan(context.Background(),
		metaengine.WithSort("priority", true),
		metaengine.WithLimit(10))
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
