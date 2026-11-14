package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/suppression"
)

var errSuppressionAudit = errors.New("suppression audit")

// runSuppressionAudit runs the full lint pipeline, collects all inline
// suppressions, cross-references them with findings, and renders an audit
// report showing each suppression's status (active, stale, unknown-rule).
// When fix is set, stale whole-line suppressions are removed from disk first
// and the report reflects the post-fix state; with dryRun the fix is only
// planned and no file changes.
func runSuppressionAudit(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
	fix bool,
	dryRun bool,
) error {
	entries, err := computeSuppressionAudit(ctx, cfg, actx)
	if err != nil {
		return err
	}

	if fix {
		fixResult := applyOrPlanFix(entries, dryRun)
		renderFixSummary(os.Stdout, fixResult, dryRun)
		entries = dropRemovedEntries(entries, fixResult.Removed)

		// Belt-and-braces for false-stale verdicts (issue #126): re-run
		// detection over the post-prune tree — a directive whose rule now
		// fires was NOT stale, and failing loudly beats leaving a live
		// finding unsuppressed. Dry-runs change no files, so only real
		// prunes need the guard.
		if !dryRun {
			if err := verifyPruneRemovedNothingLive(ctx, cfg, actx, fixResult.Removed); err != nil {
				return err
			}
		}
	}

	renderSuppressionAudit(os.Stdout, entries)

	return nil
}

// computeSuppressionAudit runs the lint pipeline and classifies every inline
// suppression in the target tree. Shared by the text and JSON doctor paths.
func computeSuppressionAudit(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
) ([]suppression.SuppressionAuditEntry, error) {
	applyConfigOverrides(cfg, actx)

	detectors := rules.RegisterAll(actx)

	result, _, err := runPipeline(ctx, cfg, detectors)
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}

	allFindings := collectFindings(result)

	goFilePaths := make([]string, 0, len(actx.GoFiles))
	for _, gf := range actx.GoFiles {
		goFilePaths = append(goFilePaths, gf.Path)
	}

	knownRuleIDs := make(map[string]bool, 200)
	for _, r := range rules.AllRules() {
		knownRuleIDs[r.ID] = true
	}

	return suppression.AuditSuppressions(goFilePaths, allFindings, knownRuleIDs), nil
}

// verifyPruneRemovedNothingLive re-runs the detection pipeline after a
// real prune and fails when any removed directive's rule fires at (or
// around) its old position: the stale verdict those removals rode on was
// wrong, and the working-tree now carries unsuppressed findings the user
// never asked for.
func verifyPruneRemovedNothingLive(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
	removed []suppression.SuppressionAuditEntry,
) error {
	if len(removed) == 0 {
		return nil
	}

	verifyCfg := *cfg
	verifyCfg.Fix = false
	verifyCfg.DryRun = false

	result, _, err := runPipeline(ctx, &verifyCfg, rules.RegisterAll(actx))
	if err != nil {
		return fmt.Errorf("prune verification re-run: %w", err)
	}

	findings := collectFindings(result)

	var revived []string
	for _, entry := range removed {
		if suppression.WouldSuppress(findings, entry.File, entry.Line, entry.Rule) {
			revived = append(revived, fmt.Sprintf("%s:%d [%s]",
				shortenPath(entry.File), entry.Line, entry.Rule))
		}
	}

	if len(revived) > 0 {
		return fmt.Errorf(
			"%w: pruned suppressions whose rules still fire — RESTORE these directives, the stale verdict was wrong: %s",
			errSuppressionAudit,
			strings.Join(revived, ", "),
		)
	}

	return nil
}

// applyOrPlanFix removes stale directives from disk, or — with dryRun — only
// plans the removal and reports what WOULD change.
func applyOrPlanFix(
	entries []suppression.SuppressionAuditEntry,
	dryRun bool,
) suppression.FixResult {
	if dryRun {
		return suppression.PlanStaleInlineSuppressions(entries)
	}

	return suppression.RemoveStaleInlineSuppressions(entries)
}

// dropRemovedEntries filters audit entries whose (file, line) was rewritten
// by the fixer, so the rendered audit reflects the post-fix file state.
func dropRemovedEntries(
	entries []suppression.SuppressionAuditEntry,
	removed []suppression.SuppressionAuditEntry,
) []suppression.SuppressionAuditEntry {
	gone := make(map[string]bool, len(removed))
	for _, e := range removed {
		gone[fmt.Sprintf("%s:%d", e.File, e.Line)] = true
	}

	kept := make([]suppression.SuppressionAuditEntry, 0, len(entries))
	for _, e := range entries {
		if !gone[fmt.Sprintf("%s:%d", e.File, e.Line)] {
			kept = append(kept, e)
		}
	}

	return kept
}

// failOnStaleSuppressions re-runs the suppression classification and returns
// a non-nil error when stale or unknown-rule inline suppressions exist. It
// backs `doctor --fail-on-stale-suppressions` (the CI gate); the audit has
// already rendered the human-readable breakdown by the time this runs.
func failOnStaleSuppressions(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
) error {
	entries, err := computeSuppressionAudit(ctx, cfg, actx)
	if err != nil {
		return err
	}

	var stale, unknown int
	for _, e := range entries {
		switch e.Status {
		case suppression.AuditStale:
			stale++
		case suppression.AuditUnknownRule:
			unknown++
		case suppression.AuditActive:
			// active suppressions are neither stale nor unknown
		}
	}

	if stale == 0 && unknown == 0 {
		return nil
	}

	return fmt.Errorf(
		"%w: %d stale and %d unknown-rule suppressions found — remove them or fix the rule IDs (see the audit above)",
		errSuppressionAudit,
		stale,
		unknown,
	)
}
