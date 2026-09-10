package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// The typed-path tests (F091 Tier 2 / F090(b)) run against the committed
// fixture module at testdata/typedfixture — a real consumer go.mod with a
// replace-based schema dependency, so packages.Load produces genuine type
// information. The in-source BuildContextFromSource harness is syntax-only
// and can never exercise the typed tier.

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

// TestV007_F090b_AttributesDotImportedRemovedSymbol pins F090(b): the bare
// `VersionedStore` reference (no qualifier, reachable only through the
// dot-import) is attributed via type info and fires V007, while the surviving
// `UpcastSourceTransform` through the same dot-import stays silent.
func TestV007_F090b_AttributesDotImportedRemovedSymbol(t *testing.T) {
	ctx := buildTypedFixtureContext(t)

	findings := ruletest.RunDetector(t, NewV007Detector(ctx))

	var dotImportWarning, bareAttribution, falsePositive int
	for _, f := range findings {
		switch {
		case strings.Contains(f.Message, "dot-import of go-cqrs-lite module"):
			dotImportWarning++
		case strings.Contains(f.Message, "VersionedStore (dot-imported from schema)"):
			bareAttribution++
		case strings.Contains(f.Message, "UpcastSourceTransform"):
			falsePositive++
		}
	}

	if dotImportWarning != 1 {
		t.Errorf("expected exactly 1 F090(a) dot-import warning, got %d", dotImportWarning)
	}
	if bareAttribution != 1 {
		t.Errorf(
			"expected exactly 1 F090(b) bare-ident attribution for VersionedStore, got %d",
			bareAttribution,
		)
	}
	if falsePositive != 0 {
		t.Errorf(
			"surviving symbol UpcastSourceTransform must not fire, got %d findings",
			falsePositive,
		)
	}
}

// TestV007_F090b_SilentWhenTypedInfoOff pins the gate: with
// --typed-info=off the bare-identifier attribution disappears (name-only
// mode cannot attribute a bare ident) — only the F090(a) import warning
// remains.
func TestV007_F090b_SilentWhenTypedInfoOff(t *testing.T) {
	ctx := buildTypedFixtureContext(t)
	ctx.TypedInfoMode = "off"

	findings := ruletest.RunDetector(t, NewV007Detector(ctx))

	for _, f := range findings {
		if strings.Contains(f.Message, "dot-imported from schema") {
			t.Errorf("F090(b) attribution fired with typed-info=off: %s", f.Message)
		}
	}

	var dotImportWarning int
	for _, f := range findings {
		if strings.Contains(f.Message, "dot-import of go-cqrs-lite module") {
			dotImportWarning++
		}
	}
	if dotImportWarning != 1 {
		t.Errorf("F090(a) import warning must survive typed-info=off, got %d", dotImportWarning)
	}
}
