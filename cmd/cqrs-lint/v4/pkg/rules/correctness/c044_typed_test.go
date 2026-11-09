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
		t.Fatalf(
			"typed fixture loaded without type info — the fixture's go.mod pins are probably unresolvable",
		)
	}

	return ctx
}

// TestC044_Fixtures pins the bad/good contract: the five leak positions
// (SQL arg, cursor key concat, map key, ActorID kind-drop, StreamRef display
// composite) fire exactly once each; the six display/benign positions
// (slog argument, log.Printf, fmt.Sprintf, bare ULID-backed EventID, nested
// display argument, ParseStreamID round-trip) stay silent.
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
		case strings.Contains(f.Snippet, "a.String()"):
			bad++
		case strings.Contains(f.Snippet, "r.String()"):
			bad++
		case strings.Contains(f.Snippet, "slog.Info(") ||
			strings.Contains(f.Snippet, "log.Printf(") ||
			strings.Contains(f.Snippet, "fmt.Sprintf(") ||
			strings.Contains(f.Snippet, "e.String()") ||
			strings.Contains(f.Snippet, "strings.ToUpper(s.String())") ||
			strings.Contains(f.Snippet, "id.ParseStreamID(s.String())"):
			good++
		default:
			t.Fatalf("unexpected C044 finding outside the fixture contract: %+v", f)
		}
	}

	if bad != 5 {
		t.Fatalf("C044 must fire on exactly the 5 leak fixtures, fired on %d: %+v", bad, findings)
	}

	if good != 0 {
		t.Fatalf("C044 must stay silent on the 6 display/benign fixtures, fired on %d", good)
	}
}

// TestC044_ActorAndRefMessages pin the boundary-specific guidance: ActorID
// points at PrefixedString, StreamRef at StreamKey.
func TestC044_ActorAndRefMessages(t *testing.T) {
	ctx := buildC044FixtureContext(t)

	findings := ruletest.RunDetector(t, NewC044Detector(ctx))

	actor, ref := false, false

	for _, f := range findings {
		if !strings.Contains(string(f.Position.File), "c044fixture.go") {
			continue
		}

		if strings.Contains(f.Snippet, "a.String()") {
			if !strings.Contains(f.Message, "PrefixedString") {
				t.Fatalf("ActorID finding must point at PrefixedString, got: %s", f.Message)
			}

			actor = true
		}

		if strings.Contains(f.Snippet, "r.String()") {
			if !strings.Contains(f.Message, "StreamKey") {
				t.Fatalf("StreamRef finding must point at StreamKey, got: %s", f.Message)
			}

			ref = true
		}
	}

	if !actor || !ref {
		t.Fatalf("expected ActorID and StreamRef findings, got actor=%v ref=%v", actor, ref)
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
