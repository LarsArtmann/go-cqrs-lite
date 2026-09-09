package performance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// The P014 typed-path tests run against the committed fixture module at
// testdata/typedfixture (same harness as the V007 F090(b) tests): a real
// consumer go.mod with a replace-based dependency, so packages.Load
// produces genuine type information. The layout fixtures live in
// layoutfixture.go.

const typedFixture = "../../../testdata/typedfixture"

func buildTypedFixtureContext(t *testing.T) *analyzer.AnalysisContext {
	t.Helper()

	// The fixture is NOT a workspace member; workspace mode would refuse to
	// load it ("directory not in go.work"). Resolve it against its own go.mod.
	t.Setenv("GOWORK", "off")

	fixture, err := filepath.Abs(typedFixture)
	if err != nil {
		t.Fatalf("abs fixture path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fixture, "go.mod")); err != nil {
		t.Fatalf("typed fixture missing (expected committed testdata): %v", err)
	}

	ctx, err := analyzer.BuildContext(fixture)
	if err != nil {
		t.Fatalf("BuildContext(typedfixture): %v", err)
	}
	if len(ctx.LoadErrors) > 0 {
		t.Fatalf("typed fixture loaded with errors: %v", ctx.LoadErrors)
	}
	if !ctx.TypedConfirmations() {
		t.Fatal("typed fixture context should have typed confirmations available (auto mode)")
	}

	return ctx
}

// TestP014_FiresOnBothPathsEngine pins the detection contract: the
// bothPathsEngine variable's ApplyLayout call fires exactly once with the
// receiver type attributed, while the legacyOnlyEngine call in the same
// file stays silent (no ApplyLayoutPlan on that receiver).
func TestP014_FiresOnBothPathsEngine(t *testing.T) {
	ctx := buildTypedFixtureContext(t)

	findings := ruletest.RunDetector(t, NewP014Detector(ctx))

	var fired, falsePositive int
	for _, f := range findings {
		switch {
		case strings.Contains(f.Message, "ApplyLayout on bothPathsEngine"):
			fired++
		default:
			falsePositive++
		}
	}

	if fired != 1 {
		t.Errorf("expected exactly 1 P014 finding for bothPathsEngine, got %d", fired)
	}
	if falsePositive != 0 {
		for _, f := range findings {
			if !strings.Contains(f.Message, "bothPathsEngine") {
				t.Errorf("unexpected finding (legacyOnly must stay silent): %s", f.Message)
			}
		}
	}
}

// TestP014_SilentWhenTypedInfoOff pins the gate: with --typed-info=off the
// rule cannot attribute receiver types, so nothing fires.
func TestP014_SilentWhenTypedInfoOff(t *testing.T) {
	ctx := buildTypedFixtureContext(t)
	ctx.TypedInfoMode = "off"

	findings := ruletest.RunDetector(t, NewP014Detector(ctx))

	if len(findings) != 0 {
		for _, f := range findings {
			t.Errorf("P014 fired with typed-info=off: %s", f.Message)
		}
	}
}
