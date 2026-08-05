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

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
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
		if reencoded, err := json.Marshal(rules); err == nil {
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

// loadParentRulesConfig walks up the directory tree from lintPath looking for
// .cqrs-lint.json files. Parent config is merged into the local config:
//   - rules.disable: union (both parent and local disables apply)
//   - rules.external-api-struct-prefixes: union (both sets apply)
//
// This implements config inheritance (L1.18) for monorepo support: a root
// .cqrs-lint.json can disable rules globally, and submodules add their own
// overrides on top.
//
// The immediate parent's config is NOT loaded (cmdguard already loaded it).
// Only ancestors beyond the current directory are consulted.
func loadParentRulesConfig(lintPath string) analyzer.RulesConfig {
	absPath, err := filepath.Abs(lintPath)
	if err != nil {
		return analyzer.RulesConfig{}
	}

	var merged analyzer.RulesConfig

	dir := filepath.Dir(absPath)
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}

		configPath := filepath.Join(parent, ".cqrs-lint.json")
		data := loadConfigFileBytes(configPath)
		if data != nil {
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
		}

		dir = parent
	}

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
