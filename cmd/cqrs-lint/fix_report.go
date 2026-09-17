package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/fix"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/suppression"
)

// runPipeline builds the pipeline configuration, creates the pipeline, and
// runs it. Per-finding fix outcomes are collected via Config.OnFixOutcome so
// --fix can report what actually happened to every fixable finding (the
// upstream go-finding issue #28 UX gap, item L1-36) instead of silently
// rewriting files.
func runPipeline(
	ctx context.Context,
	cfg *AppConfig,
	detectors []finding.Detector,
) (*pipeline.PipelineResult, []pipeline.FixOutcome, error) {
	var outcomes []pipeline.FixOutcome

	pipeConfig := pipeline.Config{
		MaxIterations:       5,
		ParallelDetectors:   true,
		GracefulDegradation: true,
		DryRun:              !cfg.Fix,
		Timeout:             5 * time.Minute,
		OnFixOutcome: func(f finding.Finding, status pipeline.FixOutcomeStatus, err error) {
			outcomes = append(outcomes, pipeline.FixOutcome{Finding: f, Status: status, Err: err})
		},
		Processors: []pipeline.FindingTransformer{
			suppression.NewSuppressionFilter(),
		},
	}

	if cfg.Fix || cfg.DryRun {
		pipeConfig.FixProviders = []pipeline.FixProvider{fix.NewCQRSFixProvider()}
	}

	pipe, err := pipeline.New(pipeConfig, cfg.Path, detectors...)
	if err != nil {
		return nil, nil, fmt.Errorf("create pipeline: %w", err)
	}

	result, err := pipe.Run(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("pipeline run: %w", err)
	}

	return result, outcomes, nil
}

// outcomeStatusOrder is the display order of the fix-outcome tally.
var outcomeStatusOrder = [...]pipeline.FixOutcomeStatus{
	pipeline.FixOutcomeApplied,
	pipeline.FixOutcomeNoChange,
	pipeline.FixOutcomeRefused,
	pipeline.FixOutcomeInvalid,
	pipeline.FixOutcomeConflict,
	pipeline.FixOutcomeFailed,
}

// printFixOutcomes writes the per-finding --fix report to stderr: a status
// tally plus one sorted line per fixable finding (status, file:line, rule,
// and the provider error for failures). It goes to stderr in every output
// format so machine-readable stdout stays parseable; --quiet silences it.
func printFixOutcomes(w io.Writer, cfg *AppConfig, outcomes []pipeline.FixOutcome) {
	if !cfg.Fix || cfg.Quiet || len(outcomes) == 0 {
		return
	}

	fmt.Fprintf(w, "Fix report: %d fixable finding(s): %s\n", len(outcomes), formatOutcomeTally(outcomes))
	for _, o := range sortFixOutcomes(outcomes) {
		printFixOutcomeLine(w, o)
	}
}

// formatOutcomeTally renders the non-zero status counts in display order.
func formatOutcomeTally(outcomes []pipeline.FixOutcome) string {
	counts := make(map[pipeline.FixOutcomeStatus]int, len(outcomeStatusOrder))
	for _, o := range outcomes {
		counts[o.Status]++
	}

	parts := make([]string, 0, len(outcomeStatusOrder))
	for _, s := range outcomeStatusOrder {
		if counts[s] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[s], s))
		}
	}

	return strings.Join(parts, ", ")
}

// printFixOutcomeLine renders one outcome: status, location, rule, and for
// failed resolutions the verbatim provider error.
func printFixOutcomeLine(w io.Writer, o pipeline.FixOutcome) {
	status := fmt.Sprintf("%-9s", o.Status)
	if o.Err != nil {
		fmt.Fprintf(w, "  %s %s:%d  %s: %v\n", status, o.Finding.Position.File, o.Finding.Position.Line, o.Finding.Rule, o.Err)
		return
	}

	fmt.Fprintf(w, "  %s %s:%d  %s\n", status, o.Finding.Position.File, o.Finding.Position.Line, o.Finding.Rule)
}

// sortFixOutcomes orders outcomes by file, line, rule, then ID so the report
// is deterministic regardless of detector scheduling.
func sortFixOutcomes(outcomes []pipeline.FixOutcome) []pipeline.FixOutcome {
	sorted := slices.Clone(outcomes)
	slices.SortFunc(sorted, func(a, b pipeline.FixOutcome) int {
		if c := strings.Compare(string(a.Finding.Position.File), string(b.Finding.Position.File)); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Finding.Position.Line, b.Finding.Position.Line); c != 0 {
			return c
		}
		if c := strings.Compare(string(a.Finding.Rule), string(b.Finding.Rule)); c != 0 {
			return c
		}

		return strings.Compare(string(a.Finding.ID), string(b.Finding.ID))
	})

	return sorted
}
