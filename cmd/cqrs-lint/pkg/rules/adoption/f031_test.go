package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestF031_ScanWithoutLimitFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func readAll(ctx context.Context, r *metaengine.TypedReader[int]) {
	_, _ = r.Scan(ctx)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF031Detector(ctx))
	ruletest.AssertRule(t, findings, "F031", 1)
}

func TestF031_WithLimitPresentStaysSilent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func readPage(ctx context.Context, r *metaengine.TypedReader[int]) {
	_, _ = r.Scan(ctx, metaengine.WithLimit(50))
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF031Detector(ctx))
	ruletest.AssertRule(t, findings, "F031", 0)
}

func TestF031_OperatorCeilingSuppresses(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func compose() {
	_, _ = metaengine.Plan(nil, metaengine.WithDefaultLimit(500))
}

func readAll(ctx context.Context, r *metaengine.TypedReader[int]) {
	_, _ = r.Scan(ctx)
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF031Detector(ctx))
	ruletest.AssertRule(t, findings, "F031", 0)
}

func TestF031_ScanSecondArgPositionWithLimitSilences(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func readFiltered(ctx context.Context, r *metaengine.TypedReader[int]) {
	_, _ = r.Scan(ctx, metaengine.WithFilter("status", metaengine.FilterEq, "open"), metaengine.WithLimit(0))
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF031Detector(ctx))
	ruletest.AssertRule(t, findings, "F031", 0)
}
