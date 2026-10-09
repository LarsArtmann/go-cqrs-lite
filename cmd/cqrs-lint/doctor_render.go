package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// renderDoctorFeatureProfile shows the resolved feature profile, annotating
// which features were pinned by config/preset vs auto-detected.
func renderDoctorFeatureProfile(w io.Writer, actx *analyzer.AnalysisContext) {
	profile := actx.FeatureProfile

	_, _ = fmt.Fprintln(w, "FEATURE PROFILE")
	_, _ = fmt.Fprintln(w, "───────────────")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprint(w, profile)
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
		_, _ = fmt.Fprint(w, m.profile)
		_, _ = fmt.Fprintln(w)
	}
}
