package adoption_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// buildScanFixtureContext loads the committed scanfixture module (real go.mod,
// metaengine replace) so Scan receivers carry genuine static types — the
// typed-path harness BuildContextFromSource cannot provide (empty types.Info).
func buildScanFixtureContext(t *testing.T) *analyzer.AnalysisContext {
	t.Helper()

	t.Setenv("GOWORK", "off")

	fixture, err := filepath.Abs("../../../testdata/scanfixture")
	if err != nil {
		t.Fatalf("abs fixture path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fixture, "go.mod")); err != nil {
		t.Fatalf("scan fixture missing (expected committed testdata): %v", err)
	}

	ctx, err := analyzer.BuildContext(fixture)
	if err != nil {
		t.Fatalf("BuildContext(scanfixture): %v", err)
	}
	if len(ctx.LoadErrors) > 0 {
		t.Fatalf("scan fixture loaded with errors: %v", ctx.LoadErrors)
	}

	return ctx
}

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

func TestF031_BufioScannerScanDoesNotFire(t *testing.T) {
	// Not parallel: buildScanFixtureContext sets GOWORK via t.Setenv.

	// Typed-path proof: real static types from the committed fixture module.
	// The bufio loop and the TypedReader Scan live in the same file — exactly
	// one finding (the reader scan) may fire, proving the bufio receiver was
	// excluded by type, not by absence of Scan calls.
	ctx := buildScanFixtureContext(t)

	findings := ruletest.RunDetector(t, adoption.NewF031Detector(ctx))
	ruletest.AssertRule(t, findings, "F031", 1)

	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding, got %d", len(findings))
	}
	if want := "main.go"; filepath.Base(string(findings[0].Position.File)) != want {
		t.Errorf("finding anchored at %s, want %s (the TypedReader scan, not the bufio loop)",
			findings[0].Position.File, want)
	}
	if line := findings[0].Position.Line; line < 30 || line > 32 {
		t.Errorf("finding anchored at line %d, want the readAll body (30-32)", line)
	}
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
