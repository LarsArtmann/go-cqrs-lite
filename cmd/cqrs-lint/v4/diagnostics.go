package main

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/larsartmann/go-finding/pipeline"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
)

func printLoadErrors(w io.Writer, errors []analyzer.PackageLoadError) {
	for _, le := range errors {
		if le.PkgPath != "" {
			_, _ = fmt.Fprintf(w, "  %s (%s):\n", le.Module, le.PkgPath)
		} else {
			_, _ = fmt.Fprintf(w, "  %s:\n", le.Module)
		}
		for _, msg := range le.Errors {
			_, _ = fmt.Fprintf(w, "    %s\n", msg)
		}
	}
}

func countModules(files []*analyzer.GoFile) int {
	seen := make(map[string]bool)
	for _, f := range files {
		seen[filepath.Dir(f.Path)] = true
	}
	return len(seen)
}

// loadRawRulesJSON reads .cqrs-lint.json from the current directory and returns
// the raw JSON of the "rules" key (for unknown-key validation). Returns nil if
// the file is absent, unreadable, or has no "rules" key — validation is
// best-effort and must never block a lint run.
func loadRawRulesJSON() []byte {
	data := loadConfigFileBytes(".cqrs-lint.json")
	if data == nil {
		return nil
	}

	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		return nil
	}

	if rules, ok := top["rules"]; ok {
		if reencoded, err := json.Marshal(rules, json.Deterministic(true)); err == nil {
			return reencoded
		}
	}

	return nil
}

// loadConfigFileBytes reads .cqrs-lint.json from the current directory and returns
// the raw bytes, with JSONC comments stripped. Returns nil if the file is
// absent or unreadable — callers are best-effort and must never block a lint run.
func loadConfigFileBytes(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return stripJSONComments(data)
}

// eachAncestorConfigFile calls fn with the .cqrs-lint.json path of every
// ancestor directory at or above the IMMEDIATE PARENT, walking upward to the
// filesystem root. Shared by the config-merge and doctor paths so both see
// the same ancestor set.
//
// An earlier version started at the grandparent, on the assumption that
// cmdguard's loader already applies the immediate parent's rules —
// empirically false: cmdguard reads the parent config for its own option
// keys but never merges cqrs-lint's rules.disable into the lint run, so
// every depth-1 module ignored root-config disables that depth-2 modules
// honored (found 2026-11-17 when the cqrs-htmx gate escalated to
// fail-on-any-finding and six depth-1 modules resurfaced findings the
// toolsdk walk — whose analyzer.LoadParentRulesConfig always included the
// immediate parent — had inherited away). Fixed in v4.22.1.
func eachAncestorConfigFile(lintPath string, fn func(configPath string)) {
	absPath, err := filepath.Abs(lintPath)
	if err != nil {
		return
	}

	for cursor := filepath.Dir(absPath); ; {
		fn(filepath.Join(cursor, ".cqrs-lint.json"))

		parent := filepath.Dir(cursor)
		if parent == cursor {
			return // reached filesystem root
		}

		cursor = parent
	}
}

// loadParentRulesConfig walks up the directory tree from lintPath looking for
// .cqrs-lint.json files. Parent config is merged into the local config:
//   - rules.disable: union (both parent and local disables apply)
//   - rules.external-api-struct-prefixes: union (both sets apply)
//
// This implements config inheritance (L1.18) for monorepo support: a root
// .cqrs-lint.json can disable rules globally, and submodules add their own
// overrides on top.
//
// The IMMEDIATE PARENT IS INCLUDED (v4.22.1): an earlier version skipped it
// on the assumption cmdguard's loader already applies that layer — false for
// cqrs-lint's rules block, which left depth-1 modules deaf to root-config
// disables (see eachAncestorConfigFile). The lint dir's own config stays
// excluded here; LoadProjectConfig/cmdguard own it.
func loadParentRulesConfig(lintPath string) analyzer.RulesConfig {
	var merged analyzer.RulesConfig

	eachAncestorConfigFile(lintPath, func(configPath string) {
		data := loadConfigFileBytes(configPath)
		if data == nil {
			return
		}

		var top struct {
			Rules analyzer.RulesConfig `json:"rules"`
		}
		if json.Unmarshal(data, &top) == nil {
			merged.Disable = append(merged.Disable, top.Rules.Disable...)
			merged.ExternalAPIStructPrefixes = append(
				merged.ExternalAPIStructPrefixes,
				top.Rules.ExternalAPIStructPrefixes...,
			)
		}
	})

	return merged
}

// validatePresetName warns if the preset name is not recognized. This catches
// typos like "prod" instead of "production" or stale names from older versions
// ("server", "full-stack") that would otherwise silently disable every preset
// override.
func validatePresetName(w io.Writer, preset analyzer.ConfigPreset) {
	if preset == "" || analyzer.IsKnownPreset(preset) {
		return
	}
	_, _ = fmt.Fprintf(
		w,
		"warning: unknown preset %q (available: %s)\n",
		preset,
		strings.Join(analyzer.ValidPresetNames(), ", "),
	)
}

// validateDisabledRuleIDs warns when the disable list references rule IDs that
// don't match any known rule. This catches typos like "C99" instead of "C009"
// or references to rules that were renamed/removed — both would silently
// disable nothing, leaving the user thinking a rule is off when it isn't.
func validateDisabledRuleIDs(w io.Writer, disabled []string) {
	if len(disabled) == 0 {
		return
	}
	for _, id := range disabled {
		id = strings.ToUpper(strings.TrimSpace(id))
		if id == "" {
			continue
		}
		if _, ok := rules.LookupRule(id); !ok {
			_, _ = fmt.Fprintf(
				w,
				"warning: disabled rule %q is not a known rule ID (typo or removed rule?)\n",
				id,
			)
		}
	}
}

// validateSeverityOverrideRuleIDs warns when a rules.severity-overrides entry
// references a rule ID that doesn't match any known rule. Same typo class as
// validateDisabledRuleIDs: the override silently does nothing. Runs after the
// parent-config merge so inherited overrides are covered; keys are sorted for
// deterministic output.
func validateSeverityOverrideRuleIDs(w io.Writer, overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}

	ids := make([]string, 0, len(overrides))
	for id := range overrides {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		id = strings.ToUpper(strings.TrimSpace(id))
		if id == "" {
			continue
		}
		if _, ok := rules.LookupRule(id); !ok {
			_, _ = fmt.Fprintf(
				w,
				"warning: severity override for %q is not a known rule ID (typo or removed rule?)\n",
				id,
			)
		}
	}
}

func printDetectorTimings(w io.Writer, snap pipeline.MetricsSnapshot) {
	if len(snap.DetectorTimes) == 0 {
		return
	}

	type detStat struct {
		name     string
		duration time.Duration
		findings int
	}

	stats := make([]detStat, 0, len(snap.DetectorTimes))
	for name, d := range snap.DetectorTimes {
		stats = append(stats, detStat{name: name, duration: d, findings: snap.FindingsFound[name]})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].duration > stats[j].duration
	})

	_, _ = fmt.Fprintln(w, "Detector timings (slowest first):")
	for _, s := range stats {
		if s.duration < time.Millisecond {
			continue
		}
		_, _ = fmt.Fprintf(
			w,
			"  %-40s %8s  %d findings\n",
			s.name,
			s.duration.Round(time.Millisecond),
			s.findings,
		)
	}
	_, _ = fmt.Fprintln(w)
}
