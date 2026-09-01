package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/suppression"
)

// runSuppressionAudit runs the full lint pipeline, collects all inline
// suppressions, cross-references them with findings, and renders an audit
// report showing each suppression's status (active, stale, unknown-rule).
// When fix is set, stale whole-line suppressions are removed from disk first
// and the report reflects the post-fix state.
func runSuppressionAudit(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
	fix bool,
) error {
	applyConfigOverrides(cfg, actx)

	detectors := rules.RegisterAll(actx)

	result, err := runPipeline(ctx, cfg, detectors)
	if err != nil {
		return fmt.Errorf("pipeline: %w", err)
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

	entries := suppression.AuditSuppressions(goFilePaths, allFindings, knownRuleIDs)

	if fix {
		fixResult := suppression.RemoveStaleInlineSuppressions(entries)
		renderFixSummary(os.Stdout, fixResult)
		entries = dropRemovedEntries(entries, fixResult.Removed)
	}

	renderSuppressionAudit(os.Stdout, entries)

	return nil
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

// renderFixSummary reports what --fix rewrote and what was left for manual
// removal.
func renderFixSummary(w io.Writer, res suppression.FixResult) {
	if len(res.Removed) == 0 && len(res.Skipped) == 0 {
		_, _ = fmt.Fprintln(
			w,
			"AUTO-FIX: no stale whole-line suppressions found — nothing removed.",
		)
		_, _ = fmt.Fprintln(w)
		return
	}

	header := fmt.Sprintf(
		"AUTO-FIX — removed %d stale suppression line(s) in %d file(s)",
		len(res.Removed),
		len(res.Files),
	)
	_, _ = fmt.Fprintln(w, header)
	_, _ = fmt.Fprintln(w, strings.Repeat("─", len(header)))

	for _, e := range res.Removed {
		_, _ = fmt.Fprintf(w, "  removed %s:%d  [%s]\n", shortenPath(e.File), e.Line, e.Rule)
	}

	if len(res.Skipped) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(
			w,
			"  %d stale suppression(s) left in place (trailing on code or unreadable) — remove manually:\n",
			len(res.Skipped),
		)
		for _, e := range res.Skipped {
			_, _ = fmt.Fprintf(w, "  %s:%d  [%s]\n", shortenPath(e.File), e.Line, e.Rule)
		}
	}

	_, _ = fmt.Fprintln(w)
}

// renderSuppressionAudit prints the audit report grouped by status.
func renderSuppressionAudit(w io.Writer, entries []suppression.SuppressionAuditEntry) {
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(w, "No inline suppressions found.")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Inline suppressions use: //cqrs-lint:ignore(RULE) reason text")
		return
	}

	var active, stale, unknown []suppression.SuppressionAuditEntry
	for _, e := range entries {
		switch e.Status {
		case suppression.AuditActive:
			active = append(active, e)
		case suppression.AuditStale:
			stale = append(stale, e)
		case suppression.AuditUnknownRule:
			unknown = append(unknown, e)
		}
	}

	total := len(entries)
	_, _ = fmt.Fprintf(w, "SUPPRESSION AUDIT — %d total\n", total)
	_, _ = fmt.Fprintln(
		w,
		strings.Repeat("─", len(fmt.Sprintf("SUPPRESSION AUDIT — %d total", total))),
	)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  Active:       %d  (suppressing a real finding)\n", len(active))
	_, _ = fmt.Fprintf(w, "  Stale:        %d  (no finding fires — safe to remove)\n", len(stale))
	_, _ = fmt.Fprintf(w, "  Unknown rule: %d  (typo or removed rule ID)\n", len(unknown))
	_, _ = fmt.Fprintln(w)

	if len(stale) > 0 {
		_, _ = fmt.Fprintln(w, "STALE SUPPRESSIONS (safe to remove)")
		_, _ = fmt.Fprintln(w, "──────────────────────────────────")
		_, _ = fmt.Fprintln(w)
		for _, e := range stale {
			renderAuditEntry(w, e)
		}
	}

	if len(unknown) > 0 {
		_, _ = fmt.Fprintln(w, "UNKNOWN-RULE SUPPRESSIONS (typo or removed rule)")
		_, _ = fmt.Fprintln(w, "─────────────────────────────────────────────────")
		_, _ = fmt.Fprintln(w)
		for _, e := range unknown {
			renderAuditEntry(w, e)
		}
	}

	if len(active) > 0 {
		_, _ = fmt.Fprintln(w, "ACTIVE SUPPRESSIONS (working correctly)")
		_, _ = fmt.Fprintln(w, "────────────────────────────────────────")
		_, _ = fmt.Fprintln(w)
		for _, e := range active {
			renderAuditEntry(w, e)
		}
	}

	if len(stale) > 0 || len(unknown) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(
			w,
			"  %d of %d suppression(s) need attention.\n",
			len(stale)+len(unknown),
			total,
		)
		_, _ = fmt.Fprintln(w, "  Remove stale suppressions and fix unknown-rule references.")
	}
}

func renderAuditEntry(w io.Writer, e suppression.SuppressionAuditEntry) {
	reason := e.Reason
	if reason == "" {
		reason = "(no reason given)"
	}

	rel := e.File
	short := shortenPath(rel)
	_, _ = fmt.Fprintf(w, "  %s:%d  [%s]  %s\n", short, e.Line, e.Rule, reason)
}

// shortenPath reduces a file path to its last 2 segments for readability.
func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}
