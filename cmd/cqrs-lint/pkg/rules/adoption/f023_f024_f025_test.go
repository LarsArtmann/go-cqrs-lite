package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestF023_ManualFilterFiresForSQLStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func filter(items []int) []int {
	var result []int
	for _, v := range items {
		if v > 10 {
			result = append(result, v)
		}
	}
	return result
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF023Detector(ctx))
	ruletest.AssertRule(t, findings, "F023", 1)
}

func TestF023_NoFindingWithMetaengine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func filter(items []int) []int {
	var result []int
	for _, v := range items {
		if v > 10 {
			result = append(result, v)
		}
	}
	_ = metaengine.Query[any, any]("q")
	return result
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF023Detector(ctx))
	ruletest.AssertRule(t, findings, "F023", 0)
}

func TestF023_NoFindingForMemoryStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func filter(items []int) []int {
	var result []int
	for _, v := range items {
		if v > 10 {
			result = append(result, v)
		}
	}
	return result
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreMemory

	findings := ruletest.RunDetector(t, adoption.NewF023Detector(ctx))
	ruletest.AssertRule(t, findings, "F023", 0)
}

func TestF023_NoFindingWithoutFilterPattern(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func process(items []int) int {
	total := 0
	for _, v := range items {
		total += v
	}
	return total
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF023Detector(ctx))
	ruletest.AssertRule(t, findings, "F023", 0)
}

func TestF024_ManualPaginationFiresForSQLStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func paginate(items []int, offset, limit int) []int {
	return items[offset : offset+limit]
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePostgres

	findings := ruletest.RunDetector(t, adoption.NewF024Detector(ctx))
	ruletest.AssertRule(t, findings, "F024", 1)
}

func TestF024_NoFindingWithMetaengine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func paginate(items []int, offset, limit int) []int {
	_ = metaengine.Query[any, any]("q")
	return items[offset : offset+limit]
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePostgres

	findings := ruletest.RunDetector(t, adoption.NewF024Detector(ctx))
	ruletest.AssertRule(t, findings, "F024", 0)
}

func TestF024_NoFindingForNonPaginationSlice(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func firstThree(items []int) []int {
	return items[:3]
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF024Detector(ctx))
	ruletest.AssertRule(t, findings, "F024", 0)
}

func TestF024_NoFindingForPebbleStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func paginate(items []int, offset, limit int) []int {
	return items[offset : offset+limit]
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePebble

	findings := ruletest.RunDetector(t, adoption.NewF024Detector(ctx))
	ruletest.AssertRule(t, findings, "F024", 0)
}

func TestF025_ManualCountFiresForSQLStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func countActive(items []bool) int {
	count := 0
	for _, active := range items {
		if active {
			count++
		}
	}
	return count
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF025Detector(ctx))
	ruletest.AssertRule(t, findings, "F025", 1)
}

func TestF025_ManualSumFiresForSQLStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func totalAmount(amounts []int) int {
	total := 0
	for _, amt := range amounts {
		total += amt
	}
	return total
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreMySQL

	findings := ruletest.RunDetector(t, adoption.NewF025Detector(ctx))
	ruletest.AssertRule(t, findings, "F025", 1)
}

func TestF025_NoFindingWithMetaengine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func totalAmount(amounts []int) int {
	total := 0
	for _, amt := range amounts {
		total += amt
	}
	_ = metaengine.Query[any, any]("q")
	return total
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, adoption.NewF025Detector(ctx))
	ruletest.AssertRule(t, findings, "F025", 0)
}

func TestF025_NoFindingForMemoryStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func countActive(items []bool) int {
	count := 0
	for _, active := range items {
		if active {
			count++
		}
	}
	return count
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreMemory

	findings := ruletest.RunDetector(t, adoption.NewF025Detector(ctx))
	ruletest.AssertRule(t, findings, "F025", 0)
}
