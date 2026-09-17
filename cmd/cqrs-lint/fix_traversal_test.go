package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
)

// TestFixStaysInsideTargetTree pins the --fix path-traversal safety
// contract (pipeline.ResolveSafePath inheritance): a finding whose position
// resolves OUTSIDE the linted tree must never be applied to disk — the file
// outside keeps its original content and the outcome records the refusal,
// while the exit semantics of runPipeline stay unchanged.
func TestFixStaysInsideTargetTree(t *testing.T) {
	t.Parallel()

	src := `package main

import "example.com/x/event"

type state struct{ N int }

func apply(state state, evt event.Event) (state, error) {
	switch evt.Type() {
	case "x.created":
		state.N++
	default:
		return state, nil
	}

	return state, nil
}
`
	actx, cleanup := analyzer.BuildContextFromTempFiles(t, map[string]string{"main.go": src})
	defer cleanup()

	// A sibling directory OUTSIDE the linted tree that happens to contain a
	// file with matching BeforeCode content — the traversal bait.
	tree := filepath.Dir(actx.GoFiles[0].Path)
	outside := filepath.Join(filepath.Dir(tree), "outside.txt")
	if err := os.WriteFile(outside, []byte("DO NOT TOUCH"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outside) // tree itself is temp-managed

	traversal := finding.NewBuilder(
		"C003", "cqrs-lint",
		"traversal probe",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath("../outside.txt"), 1, 1),
	).
		WithFixStrategy(finding.FixStrategyDirect).
		WithBeforeCode("DO NOT TOUCH").
		WithAfterCode("PWNED").
		BuildOrDefault()

	detector := finding.NamedDetectorFunc(
		"traversal-probe",
		func(_ context.Context) ([]finding.Finding, error) {
			return []finding.Finding{traversal}, nil
		},
	)

	cfg := &AppConfig{Fix: true, Path: tree}
	_, outcomes, err := runPipeline(context.Background(), cfg, []finding.Detector{detector})
	if err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	after, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}

	if string(after) != "DO NOT TOUCH" {
		t.Fatalf("traversal: file outside the tree was rewritten:\n%s", after)
	}

	for _, o := range outcomes {
		if o.Status == pipeline.FixOutcomeApplied {
			t.Fatalf("traversal finding reported as applied: %+v", o)
		}

		if strings.Contains(string(o.Finding.Position.File), "outside") &&
			o.Status == pipeline.FixOutcomeApplied {
			t.Fatal("unreachable")
		}
	}
}
