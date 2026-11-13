package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/suppression"
)

// renderDoctorFeatureProfile shows the resolved feature profile, annotating
// which features were pinned by config/preset vs auto-detected.
func renderDoctorFeatureProfile(w io.Writer, actx *analyzer.AnalysisContext) {
	profile := actx.FeatureProfile

	_, _ = fmt.Fprintln(w, "FEATURE PROFILE")
	_, _ = fmt.Fprintln(w, "───────────────")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprint(w, &profile)
	_, _ = fmt.Fprintln(w)

	// Show config overrides if any features were explicitly pinned
	cf := actx.FeatureProfile.ToConfigFeatures()
	hasOverrides := cf.Store != nil || cf.CommandFlow != nil || cf.Server != nil ||
		cf.SoftDelete != nil || cf.Tracing != nil || cf.Snapshot != nil ||
		cf.Domain != nil || cf.Monetary != nil || cf.Transport != nil || cf.ServerLocal != nil ||
		cf.AsyncBus != nil

	if hasOverrides {
		_, _ = fmt.Fprintln(
			w,
			"  Note: some features are pinned by config/preset (see EFFECTIVE SETTINGS above).",
		)
		_, _ = fmt.Fprintln(
			w,
			"        Unpinned features were auto-detected from source code analysis.",
		)
		_, _ = fmt.Fprintln(w)
	}

	// Cross-render hint: doctor already surfaces the composition signal; the
	// scorecard acts on it (persistence rows credited, Modernity grade).
	if profile.HasSystemComposition {
		_, _ = fmt.Fprintln(
			w,
			"  Hint: system.New composition detected — the scorecard credits persistence rows",
		)
		_, _ = fmt.Fprintln(
			w,
			"        as used via this wiring (see `cqrs-lint scorecard`).",
		)
		_, _ = fmt.Fprintln(w)
	}
}

// renderDoctorPerModuleProfiles shows each module's profile in multi-module workspaces.
func renderDoctorPerModuleProfiles(w io.Writer, actx *analyzer.AnalysisContext) {
	if len(actx.FeatureProfiles) <= 1 {
		return
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "PER-MODULE PROFILES (%d modules)\n", len(actx.FeatureProfiles))
	_, _ = fmt.Fprintln(w, "─────────────────────────────────")
	_, _ = fmt.Fprintln(w)

	type modProfile struct {
		dir     string
		profile analyzer.FeatureProfile
	}

	mods := make([]modProfile, 0, len(actx.FeatureProfiles))
	for dir, p := range actx.FeatureProfiles {
		mods = append(mods, modProfile{dir, p})
	}

	sort.Slice(mods, func(i, j int) bool {
		// Length order groups nested modules under their parents; the name
		// tie-break makes the output deterministic (map iteration is not).
		if len(mods[i].dir) != len(mods[j].dir) {
			return len(mods[i].dir) < len(mods[j].dir)
		}
		return mods[i].dir < mods[j].dir
	})

	for _, m := range mods {
		rel := m.dir
		if rel == "" {
			rel = "(root)"
		}

		_, _ = fmt.Fprintf(w, "=== %s ===\n", rel)
		_, _ = fmt.Fprint(w, &m.profile)
		_, _ = fmt.Fprintln(w)
	}
}

// renderFixSummary reports what --prune-suppressions rewrote (or, in dry-run,
// WOULD rewrite) and what was left for manual removal.
func renderFixSummary(w io.Writer, res suppression.FixResult, dryRun bool) {
	if len(res.Removed) == 0 && len(res.Skipped) == 0 {
		if dryRun {
			_, _ = fmt.Fprintln(
				w,
				"AUTO-FIX (dry-run): no stale whole-line suppressions found — nothing would be removed.",
			)
			_, _ = fmt.Fprintln(w)

			return
		}

		_, _ = fmt.Fprintln(
			w,
			"AUTO-FIX: no stale whole-line suppressions found — nothing removed.",
		)
		_, _ = fmt.Fprintln(w)

		return
	}

	action := "removed"
	prefix := ""

	if dryRun {
		action = "would remove"
		prefix = "DRY-RUN — no files changed. "
	}

	header := fmt.Sprintf(
		"AUTO-FIX — %s %d stale suppression line(s) in %d file(s)",
		action,
		len(res.Removed),
		len(res.Files),
	)
	_, _ = fmt.Fprintln(w, header)
	_, _ = fmt.Fprintln(w, strings.Repeat("─", len(header)))
	_, _ = fmt.Fprintln(w)

	if dryRun {
		_, _ = fmt.Fprintf(w, "  %s\n", prefix)
	}

	for _, e := range res.Removed {
		_, _ = fmt.Fprintf(w, "  %s %s:%d  [%s]\n", action, shortenPath(e.File), e.Line, e.Rule)
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
		_, _ = fmt.Fprintln(w, "──────────────────────────────")
		_, _ = fmt.Fprintln(w)
		for _, e := range stale {
			renderAuditEntry(w, e)
		}
	}

	if len(unknown) > 0 {
		_, _ = fmt.Fprintln(w, "UNKNOWN-RULE SUPPRESSIONS (typo or removed rule)")
		_, _ = fmt.Fprintln(w, "─────────────────────────────────")
		_, _ = fmt.Fprintln(w)
		for _, e := range unknown {
			renderAuditEntry(w, e)
		}
	}

	if len(active) > 0 {
		_, _ = fmt.Fprintln(w, "ACTIVE SUPPRESSIONS (working correctly)")
		_, _ = fmt.Fprintln(w, "──────────────────────────")
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
