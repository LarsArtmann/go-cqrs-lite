package correctness

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// The C044 typed-path tests run against the committed fixture module at
// testdata/typedfixture (same harness as P014/V007): a real consumer go.mod
// with resolvable pins, so packages.Load produces genuine type information —
// the rule resolves .String() receivers through TypesInfo.

const c044TypedFixture = "../../../testdata/typedfixture"

func buildC044FixtureContext(t *testing.T) *analyzer.AnalysisContext {
	t.Helper()

	t.Setenv("GOWORK", "off")

	ctx, err := analyzer.BuildContext(c044TypedFixture)
	if err != nil {
		t.Fatalf("BuildContext(typedfixture): %v", err)
	}

	if len(ctx.LoadErrors) > 0 {
		t.Fatalf("typed fixture loaded with errors: %v", ctx.LoadErrors)
	}

	if !ctx.TypedConfirmations() {
		t.Fatalf("typed fixture loaded without type info — the fixture's go.mod pins are probably unresolvable")
	}

	return ctx
}

// TestC044_Fixtures pins the 3-good/3-bad contract: the three leak positions
// (SQL arg, cursor key concat, map key) fire exactly once each; the three
// display positions (slog argument, log.Printf, fmt.Sprintf) stay silent.
func TestC044_Fixtures(t *testing.T) {
	ctx := buildC044FixtureContext(t)

	findings := ruletest.RunDetector(t, NewC044Detector(ctx))

	bad := 0

	good := 0

	for _, f := range findings {
		if !strings.Contains(string(f.Position.File), "c044fixture.go") {
			continue
		}

		switch {
		case strings.Contains(f.Snippet, "db.Query("):
			bad++
		case strings.Contains(f.Snippet, `"u:" + s.String()`):
			bad++
		case strings.Contains(f.Snippet, "projection[s.String()]"):
			bad++
		case strings.Contains(f.Snippet, "slog.Info(") ||
			strings.Contains(f.Snippet, "log.Printf(") ||
			strings.Contains(f.Snippet, "fmt.Sprintf("):
			good++
		default:
			t.Fatalf("unexpected C044 finding outside the fixture contract: %+v", f)
		}
	}

	if bad != 3 {
		t.Fatalf("C044 must fire on exactly the 3 leak fixtures, fired on %d: %+v", bad, findings)
	}

	if good != 0 {
		t.Fatalf("C044 must stay silent on the 3 display fixtures, fired on %d", good)
	}
}

// TestC044_WarnSeverityFirst pins the warn-first contract (the plan's G4
// pattern: hard severity waits for v5).
func TestC044_WarnSeverityFirst(t *testing.T) {
	ctx := buildC044FixtureContext(t)

	findings := ruletest.RunDetector(t, NewC044Detector(ctx))

	for _, f := range findings {
		if !strings.Contains(string(f.Position.File), "c044fixture.go") {
			continue
		}

		if f.Severity != "warning" {
			t.Fatalf("C044 must be warn-first, got severity %q", f.Severity)
		}
	}
}
