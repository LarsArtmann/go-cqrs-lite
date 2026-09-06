package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding/pipeline"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/fix"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
)

// TestFixE2E_C003AppliesAtReportedOccurrence drives the real go-finding
// pipeline (triage -> provider -> byte-level applier) against a fixture on
// disk and asserts the edit lands on the reported occurrence — and only
// there. It exists because a provider/detector mismatch used to surface as
// a SILENT no-op: C003 anchored its fix data to the function declaration
// while BeforeCode lived on the default-case return line, so the provider's
// line-scoped match correctly refused and `--fix` changed nothing without
// any diagnostic.
func TestFixE2E_C003AppliesAtReportedOccurrence(t *testing.T) {
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

	file := actx.GoFiles[0].Path
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	if strings.Count(string(before), "return state, nil") != 2 {
		t.Fatalf(
			"fixture changed: expected 2 occurrences of the target text, got %d",
			strings.Count(string(before), "return state, nil"),
		)
	}

	pipe, err := pipeline.New(pipeline.Config{
		MaxIterations: 5,
		DryRun:        false,
		FixProviders:  []pipeline.FixProvider{fix.NewCQRSFixProvider()},
	}, filepath.Dir(file), correctness.NewC003Detector(actx))
	if err != nil {
		t.Fatalf("create pipeline: %v", err)
	}

	res, err := pipe.Run(context.Background())
	if err != nil {
		t.Fatalf("pipeline run: %v", err)
	}

	if len(res.Iterations) == 0 || res.Iterations[0].Applied != 1 {
		t.Fatalf("expected exactly 1 applied fix, got %+v", res.Iterations)
	}

	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("reread fixture: %v", err)
	}

	if !strings.Contains(
		string(after),
		`return state, fmt.Errorf("fold: unknown event type: %s", evt.Type())`,
	) {
		t.Fatal("default case was not rewritten to return the unknown-event error")
	}

	if strings.Count(string(after), "return state, nil") != 1 {
		t.Fatal(
			"the final return outside the switch must stay untouched — the fix edited a wrong occurrence",
		)
	}
}
