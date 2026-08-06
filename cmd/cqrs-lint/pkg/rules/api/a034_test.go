package api_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestA034_ExecuteUntypedFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	_, _ = metaengine.Execute(context.Background(), nil, "query", nil)
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, api.NewA034Detector(ctx))
	ruletest.AssertRule(t, findings, "A034", 1)
}

func TestA034_NoFindingForExecuteTyped(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func _() {
	_, _ = metaengine.ExecuteTyped[any, any](context.Background(), nil, nil)
}
`,
	})
	ctx.FeatureProfile.HasMetaengine = true

	findings := ruletest.RunDetector(t, api.NewA034Detector(ctx))
	ruletest.AssertRule(t, findings, "A034", 0)
}

func TestA034_NoFindingWithoutMetaengine(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func _() {
}
`,
	})

	findings := ruletest.RunDetector(t, api.NewA034Detector(ctx))
	ruletest.AssertRule(t, findings, "A034", 0)
}
