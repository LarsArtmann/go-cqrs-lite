package main

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
)

func outcomeFixture() []pipeline.FixOutcome {
	return []pipeline.FixOutcome{
		{
			Finding: finding.Finding{
				ID:       "cqrs-lint:C010:b.go:7",
				Rule:     "C010",
				Position: finding.Position{File: "b.go", Line: 7},
			},
			Status: pipeline.FixOutcomeConflict,
		},
		{
			Finding: finding.Finding{
				ID:       "cqrs-lint:C003:a.go:42",
				Rule:     "C003",
				Position: finding.Position{File: "a.go", Line: 42},
			},
			Status: pipeline.FixOutcomeApplied,
		},
		{
			Finding: finding.Finding{
				ID:       "cqrs-lint:D006:c.go:1",
				Rule:     "D006",
				Position: finding.Position{File: "c.go", Line: 1},
			},
			Status: pipeline.FixOutcomeFailed,
			Err:    errors.New("provider boom"),
		},
	}
}

// TestPrintFixOutcomesRendersPerFinding pins the exact report format: tally
// first, then one sorted line per finding with the provider error verbatim.
func TestPrintFixOutcomesRendersPerFinding(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printFixOutcomes(&buf, &AppConfig{Fix: true}, outcomeFixture())

	want := strings.Join([]string{
		"Fix report: 3 fixable finding(s): 1 applied, 1 conflict, 1 failed",
		"  applied   a.go:42  C003",
		"  conflict  b.go:7  C010",
		"  failed    c.go:1  D006: provider boom",
		"",
	}, "\n")
	if got := buf.String(); got != want {
		t.Fatalf("report mismatch\ngot:\n%q\nwant:\n%q", got, want)
	}
}

// TestPrintFixOutcomesSkipsQuietNonFixAndEmpty: no report when --fix did not
// run (DryRun / plain detect), when --quiet was passed, or when nothing was
// fixable.
func TestPrintFixOutcomesSkipsQuietNonFixAndEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		cfg      AppConfig
		outcomes []pipeline.FixOutcome
	}{
		{"dry-run (no --fix)", AppConfig{}, outcomeFixture()},
		{"quiet", AppConfig{Fix: true, Quiet: true}, outcomeFixture()},
		{"nothing fixable", AppConfig{Fix: true}, nil},
	}

	for _, tc := range cases {
		var buf bytes.Buffer
		printFixOutcomes(&buf, &tc.cfg, tc.outcomes)
		if buf.Len() != 0 {
			t.Fatalf("%s: expected no output, got %q", tc.name, buf.String())
		}
	}
}

// TestCollectFixOutcomesKeepsFirstOutcomePerFinding: re-detection after an
// applied fix re-fires the same finding (stale AST snapshot) and the provider
// then refuses — only the FIRST outcome per finding ID is an outcome; the
// repeats are artifacts.
func TestCollectFixOutcomesKeepsFirstOutcomePerFinding(t *testing.T) {
	t.Parallel()

	f := finding.Finding{
		ID:       "cqrs-lint:C003:a.go:42",
		Rule:     "C003",
		Position: finding.Position{File: "a.go", Line: 42},
	}

	var out []pipeline.FixOutcome
	collect := collectFixOutcomes(&out)

	collect(f, pipeline.FixOutcomeApplied, nil)
	collect(f, pipeline.FixOutcomeRefused, nil)
	collect(f, pipeline.FixOutcomeFailed, errors.New("late boom"))

	if len(out) != 1 || out[0].Status != pipeline.FixOutcomeApplied {
		t.Fatalf("expected single applied outcome, got %+v", out)
	}

	other := f
	other.ID = "cqrs-lint:C010:a.go:9"
	other.Rule = "C010"
	other.Position.Line = 9
	collect(other, pipeline.FixOutcomeRefused, nil)

	if len(out) != 2 {
		t.Fatalf("distinct finding must be recorded, got %+v", out)
	}
}

// TestRunPipelineCollectsFixOutcomes drives the real pipeline over a fixable
// C003 fixture and asserts the collector records exactly one applied outcome
// with the finding's rule — the wiring behind the --fix report.
func TestRunPipelineCollectsFixOutcomes(t *testing.T) {
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

	cfg := &AppConfig{Fix: true, Path: filepath.Dir(actx.GoFiles[0].Path)}
	_, outcomes, err := runPipeline(
		context.Background(), cfg,
		[]finding.Detector{correctness.NewC003Detector(actx)},
	)
	if err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	if len(outcomes) != 1 {
		t.Fatalf("expected exactly 1 fix outcome, got %d: %+v", len(outcomes), outcomes)
	}

	if outcomes[0].Status != pipeline.FixOutcomeApplied {
		t.Fatalf("expected applied, got %s (err=%v)", outcomes[0].Status, outcomes[0].Err)
	}

	if string(outcomes[0].Finding.Rule) != "C003" {
		t.Fatalf("expected rule C003, got %s", outcomes[0].Finding.Rule)
	}
}

// TestRunPipelineDryRunEmitsNoOutcomes: without --fix the pipeline never
// enters fix application, so no outcomes may be recorded or reported.
func TestRunPipelineDryRunEmitsNoOutcomes(t *testing.T) {
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

	cfg := &AppConfig{Fix: false, Path: filepath.Dir(actx.GoFiles[0].Path)}
	_, outcomes, err := runPipeline(
		context.Background(), cfg,
		[]finding.Detector{correctness.NewC003Detector(actx)},
	)
	if err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	if len(outcomes) != 0 {
		t.Fatalf("dry run must emit no fix outcomes, got %+v", outcomes)
	}
}
