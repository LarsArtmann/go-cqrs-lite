package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
)

// --- F018: FilterOn without FilterOnField ---

func TestF018_FilterOnFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.Query[any, any]("q",
		metaengine.FilterOn(func(v any) bool { return true }),
	)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF018Detector(ctx))
	ruletest.AssertRule(t, findings, "F018", 1)
}

func TestF018_NoFindingWithFilterOnField(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.Query[any, any]("q",
		metaengine.FilterOnField("status", metaengine.FilterEq, "open"),
	)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF018Detector(ctx))
	ruletest.AssertRule(t, findings, "F018", 0)
}

func TestF018_NoFindingWithoutMetaengine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
	_ = FilterOn(func(v any) bool { return true })
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF018Detector(ctx))
	ruletest.AssertRule(t, findings, "F018", 0)
}

// --- F019: Missing Volume hint ---

func TestF019_MissingVolumeFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.Query[any, any]("q",
		metaengine.OnTyped("evt", nil, func(e any) (any, any) { return nil, nil }),
	)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF019Detector(ctx))
	ruletest.AssertRule(t, findings, "F019", 1)
}

func TestF019_NoFindingWithVolume(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.Query[any, any]("q",
		metaengine.OnTyped("evt", nil, func(e any) (any, any) { return nil, nil }),
		metaengine.Volume(100000),
	)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF019Detector(ctx))
	ruletest.AssertRule(t, findings, "F019", 0)
}

// --- F020: SortOn without SortOnField ---

func TestF020_SortOnFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.Query[any, any]("q",
		metaengine.SortOn(func(a, b any) bool { return true }),
	)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF020Detector(ctx))
	ruletest.AssertRule(t, findings, "F020", 1)
}

func TestF020_NoFindingWithSortOnField(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.Query[any, any]("q",
		metaengine.SortOnField("created_at", false),
	)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF020Detector(ctx))
	ruletest.AssertRule(t, findings, "F020", 0)
}

// --- F021: Write amplification (5+ folds) ---

func TestF021_WriteAmplificationFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.Query[any, any]("q",
		metaengine.OnTyped("e1", nil, f),
		metaengine.OnTyped("e2", nil, f),
		metaengine.OnTyped("e3", nil, f),
		metaengine.OnTyped("e4", nil, f),
		metaengine.OnTyped("e5", nil, f),
	)
}

func f(e any) (any, any) { return nil, nil }
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF021Detector(ctx))
	ruletest.AssertRule(t, findings, "F021", 1)
}

func TestF021_NoFindingWithFewFolds(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"

func _() {
	_ = metaengine.Query[any, any]("q",
		metaengine.OnTyped("e1", nil, f),
		metaengine.OnTyped("e2", nil, f),
	)
}

func f(e any) (any, any) { return nil, nil }
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF021Detector(ctx))
	ruletest.AssertRule(t, findings, "F021", 0)
}
