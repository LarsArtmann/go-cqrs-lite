package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestF022_ManualSortFiresForSQLStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "sort"

func _() {
	var items []string
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF022Detector(ctx))
	ruletest.AssertRule(t, findings, "F022", 1)
}

func TestF022_NoFindingWithMetaengine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"sort"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	var items []string
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
	_ = metaengine.Query[any, any]("q")
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF022Detector(ctx))
	ruletest.AssertRule(t, findings, "F022", 0)
}

func TestF022_NoFindingForMemoryStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "sort"

func _() {
	var items []string
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreMemory

	findings := ruletest.RunDetector(t, adoption.NewF022Detector(ctx))
	ruletest.AssertRule(t, findings, "F022", 0)
}

func TestF022_NoFindingWithoutSortCalls(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	items := []string{"a", "b", "c"}
	_ = items
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF022Detector(ctx))
	ruletest.AssertRule(t, findings, "F022", 0)
}

func TestF022_SuppressedByMetaengineSubPackage(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"sort"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
)

func _() {
	var items []string
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
	_ = pebbleengine.NewPebbleEngine
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF022Detector(ctx))
	ruletest.AssertRule(t, findings, "F022", 0)
}

func TestF022_SortSlicesDetected(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "slices"

func _() {
	type item struct{ priority int }
	items := []item{{3}, {1}, {2}}
	slices.SortFunc(items, func(a, b item) int { return a.priority - b.priority })
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePostgres

	findings := ruletest.RunDetector(t, adoption.NewF022Detector(ctx))
	ruletest.AssertRule(t, findings, "F022", 1)
}

func TestF022_NoFindingForPebbleStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "sort"

func _() {
	var items []string
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePebble

	findings := ruletest.RunDetector(t, adoption.NewF022Detector(ctx))
	ruletest.AssertRule(t, findings, "F022", 0)
}
