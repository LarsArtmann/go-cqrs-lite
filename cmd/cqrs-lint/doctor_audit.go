package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/suppression"
)

// runSuppressionAudit runs the full lint pipeline, collects all inline
// suppressions, cross-references them with findings, and renders an audit
// report showing each suppression's status (active, stale, unknown-rule).
func runSuppressionAudit(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
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
	renderSuppressionAudit(os.Stdout, entries)

	return nil
}

// renderSuppressionAudit prints the audit report grouped by status.
func renderSuppressionAudit(w io.Writer, entries []suppression.SuppressionAuditEntry) {
	if len(entries) == 0 {
		fmt.Fprintln(w, "No inline suppressions found.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Inline suppressions use: //cqrs-lint:ignore(RULE) reason text")
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
	fmt.Fprintf(w, "SUPPRESSION AUDIT — %d total\n", total)
	fmt.Fprintln(w, strings.Repeat("─", len(fmt.Sprintf("SUPPRESSION AUDIT — %d total", total))))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Active:       %d  (suppressing a real finding)\n", len(active))
	fmt.Fprintf(w, "  Stale:        %d  (no finding fires — safe to remove)\n", len(stale))
	fmt.Fprintf(w, "  Unknown rule: %d  (typo or removed rule ID)\n", len(unknown))
	fmt.Fprintln(w)

	if len(stale) > 0 {
		fmt.Fprintln(w, "STALE SUPPRESSIONS (safe to remove)")
		fmt.Fprintln(w, "──────────────────────────────────")
		fmt.Fprintln(w)
		for _, e := range stale {
			renderAuditEntry(w, e)
		}
	}

	if len(unknown) > 0 {
		fmt.Fprintln(w, "UNKNOWN-RULE SUPPRESSIONS (typo or removed rule)")
		fmt.Fprintln(w, "─────────────────────────────────────────────────")
		fmt.Fprintln(w)
		for _, e := range unknown {
			renderAuditEntry(w, e)
		}
	}

	if len(active) > 0 {
		fmt.Fprintln(w, "ACTIVE SUPPRESSIONS (working correctly)")
		fmt.Fprintln(w, "────────────────────────────────────────")
		fmt.Fprintln(w)
		for _, e := range active {
			renderAuditEntry(w, e)
		}
	}

	if len(stale) > 0 || len(unknown) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %d of %d suppression(s) need attention.\n", len(stale)+len(unknown), total)
		fmt.Fprintln(w, "  Remove stale suppressions and fix unknown-rule references.")
	}
}

func renderAuditEntry(w io.Writer, e suppression.SuppressionAuditEntry) {
	reason := e.Reason
	if reason == "" {
		reason = "(no reason given)"
	}

	rel := e.File
	short := shortenPath(rel)
	fmt.Fprintf(w, "  %s:%d  [%s]  %s\n", short, e.Line, e.Rule, reason)
}

// shortenPath reduces a file path to its last 2 segments for readability.
func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}
