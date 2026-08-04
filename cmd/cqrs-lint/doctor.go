package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

func setupDoctorCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"doctor",
		cmdguard.NoFlags{},
		func(_ context.Context, cfg *AppConfig, _ cmdguard.NoFlags) error {
			actx, err := analyzer.BuildContext(cfg.Path)
			if err != nil {
				return fmt.Errorf("load packages: %w", err)
			}

			renderDoctorLoadErrors(actx)

			renderDoctorConfigFile(cfg)

			// Resolve all overrides so the rest of the output shows the
			// EFFECTIVE configuration, not just the raw detected values.
			applyConfigOverrides(cfg, actx)

			renderDoctorPreset(cfg)
			renderDoctorEffectiveSettings(cfg)
			renderDoctorFeatureProfile(actx)
			renderDoctorPerModuleProfiles(actx)
			renderDoctorSuggestedConfig(cfg, actx)
			renderDoctorSuppressions(actx)

			return nil
		},
		cmdguard.WithShort(
			"Show the project's full resolved cqrs-lint configuration and detected profile",
		),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "doctor", cmd, err)
}

// renderDoctorLoadErrors prints package-loading failures if any were detected.
func renderDoctorLoadErrors(actx *analyzer.AnalysisContext) {
	if len(actx.LoadErrors) == 0 {
		return
	}

	fmt.Fprintln(
		os.Stderr,
		"WARNING: package loading was partial; the profile below may be incomplete or misleading.",
	)
	for _, le := range actx.LoadErrors {
		if le.PkgPath != "" {
			fmt.Fprintf(os.Stderr, "  %s (%s):\n", le.Module, le.PkgPath)
		} else {
			fmt.Fprintf(os.Stderr, "  %s:\n", le.Module)
		}
		for _, msg := range le.Errors {
			fmt.Fprintf(os.Stderr, "    %s\n", msg)
		}
	}
	fmt.Fprintln(os.Stderr)
}

// renderDoctorConfigFile shows the raw .cqrs-lint.json content (if present),
// including parent configs found in ancestor directories.
func renderDoctorConfigFile(cfg *AppConfig) {
	fmt.Println("CONFIG FILE")
	fmt.Println("───────────")

	configPath := filepath.Join(cfg.Path, ".cqrs-lint.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		absPath, _ := filepath.Abs(configPath)
		fmt.Printf("  Path:    %s\n", absPath)
		fmt.Println("  Status:  NOT FOUND")
		fmt.Println()
		fmt.Println("  No .cqrs-lint.json found. Run 'cqrs-lint init' to create one,")
		fmt.Println("  or 'cqrs-lint explain' to see all available options.")
		fmt.Println()
		fmt.Println()
		return
	}

	absPath, _ := filepath.Abs(configPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	fmt.Printf("  Path:    %s\n", absPath)
	fmt.Printf("  Status:  Found (%d lines, %d bytes)\n", len(lines), len(data))
	fmt.Println()
	fmt.Println("  Content:")

	for _, line := range lines {
		fmt.Printf("    %s\n", line)
	}

	// Show parent configs (monorepo inheritance)
	parents := findParentConfigs(cfg.Path)
	if len(parents) > 0 {
		fmt.Println()
		fmt.Println("  Parent configs (merged for rule disables):")
		for _, p := range parents {
			rel, err := filepath.Rel(cfg.Path, p)
			if err != nil {
				rel = p
			}
			fmt.Printf("    %s\n", rel)
		}
	}

	fmt.Println()
	fmt.Println()
}

// findParentConfigs walks up the directory tree and returns paths to all
// .cqrs-lint.json files found in ancestor directories.
func findParentConfigs(lintPath string) []string {
	absPath, err := filepath.Abs(lintPath)
	if err != nil {
		return nil
	}

	var found []string

	dir := filepath.Dir(absPath)
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		configPath := filepath.Join(parent, ".cqrs-lint.json")
		if _, err := os.Stat(configPath); err == nil {
			found = append(found, configPath)
		}

		dir = parent
	}

	return found
}

// renderDoctorPreset shows the active preset with its description and what it provides.
func renderDoctorPreset(cfg *AppConfig) {
	fmt.Println("ACTIVE PRESET")
	fmt.Println("─────────────")

	if cfg.Preset == "" {
		fmt.Println("  (none)")
		fmt.Println()
		fmt.Println("  No preset active. All features are auto-detected.")
		fmt.Println("  See 'cqrs-lint explain' for available presets.")
		fmt.Println()
		fmt.Println()
		return
	}

	desc := presetDescriptions[cfg.Preset]
	presetDef := analyzer.ResolvePresetDefinition(cfg.Preset)

	fmt.Printf("  %s\n", cfg.Preset)
	if desc != "" {
		fmt.Printf("    %s\n", desc)
	}

	if f := formatConfigFeatures(presetDef.Features); f != "" {
		fmt.Printf("    Features pinned:  %s\n", f)
	}

	if len(presetDef.Rules.Disable) > 0 {
		fmt.Printf("    Rules disabled:   %s\n", strings.Join(presetDef.Rules.Disable, ", "))
	}

	if presetDef.MinSeverity != "" {
		fmt.Printf("    Severity floor:   %s\n", presetDef.MinSeverity)
	}

	fmt.Println()
	fmt.Println()
}

// renderDoctorEffectiveSettings shows the resolved configuration after all
// overrides (preset, config file, auto-detection) are applied.
func renderDoctorEffectiveSettings(cfg *AppConfig) {
	fmt.Println("EFFECTIVE SETTINGS")
	fmt.Println("──────────────────")
	fmt.Println()

	presetDef := analyzer.ResolvePresetDefinition(cfg.Preset)
	effectiveSev := resolveMinSeverity(presetDef.MinSeverity, cfg.MinSeverity)
	sevSource := "default"
	if presetDef.MinSeverity != "" && effectiveSev == presetDef.MinSeverity {
		sevSource = "preset floor"
	} else if effectiveSev != "info" {
		sevSource = "config"
	}

	fmt.Printf("  min-severity:    %s  (%s)\n", effectiveSev, sevSource)
	fmt.Printf("  min-confidence:  %s\n", cfg.MinConfidence)
	fmt.Printf("  format:          %s\n", cfg.Format)
	fmt.Printf("  color:           %s\n", cfg.Color)

	if cfg.Preset != "" {
		fmt.Printf("  preset:          %s\n", cfg.Preset)
	}

	// Disabled rules with source breakdown
	totalRules := len(rules.AllRules())
	disabledCount := len(cfg.Rules.Disable)
	activeCount := totalRules - disabledCount
	fmt.Printf("  rules:           %d total, %d active, %d disabled\n",
		totalRules, activeCount, disabledCount)

	if disabledCount > 0 {
		presetDisabled := presetDef.Rules.Disable
		presetSet := make(map[string]bool, len(presetDisabled))
		for _, r := range presetDisabled {
			presetSet[r] = true
		}

		var fromPreset, fromConfig []string
		for _, r := range cfg.Rules.Disable {
			if presetSet[r] {
				fromPreset = append(fromPreset, r)
			} else {
				fromConfig = append(fromConfig, r)
			}
		}

		if len(fromPreset) > 0 {
			fmt.Printf("    from preset:   %s\n", strings.Join(fromPreset, ", "))
		}
		if len(fromConfig) > 0 {
			fmt.Printf("    from config:   %s\n", strings.Join(fromConfig, ", "))
		}
	}

	// Rules overrides
	if len(cfg.Rules.ExternalAPIStructPrefixes) > 0 {
		fmt.Printf("  external-api-struct-prefixes: %s\n",
			strings.Join(cfg.Rules.ExternalAPIStructPrefixes, ", "))
	}
	if len(cfg.Rules.IgnoreFloatFields) > 0 {
		fmt.Printf("  c008-ignore-fields: %s\n",
			strings.Join(cfg.Rules.IgnoreFloatFields, ", "))
	}
	if len(cfg.Rules.IgnoreStructs) > 0 {
		fmt.Printf("  c008-ignore-structs: %s\n",
			strings.Join(cfg.Rules.IgnoreStructs, ", "))
	}

	// Health config
	if cfg.Health.InfoCap != 0 {
		fmt.Printf("  health.info-cap: %d\n", cfg.Health.InfoCap)
	}

	fmt.Println()
	fmt.Println()
}

// renderDoctorFeatureProfile shows the resolved feature profile, annotating
// which features were pinned by config/preset vs auto-detected.
func renderDoctorFeatureProfile(actx *analyzer.AnalysisContext) {
	profile := actx.FeatureProfile

	fmt.Println("FEATURE PROFILE")
	fmt.Println("───────────────")
	fmt.Println()
	fmt.Print(profile)
	fmt.Println()

	// Show config overrides if any features were explicitly pinned
	cf := actx.FeatureProfile.ToConfigFeatures()
	hasOverrides := cf.Store != nil || cf.CommandFlow != nil || cf.Server != nil ||
		cf.SoftDelete != nil || cf.Tracing != nil || cf.Snapshot != nil ||
		cf.Domain != nil || cf.Transport != nil || cf.ServerLocal != nil || cf.AsyncBus != nil

	if hasOverrides {
		fmt.Println(
			"  Note: some features are pinned by config/preset (see EFFECTIVE SETTINGS above).",
		)
		fmt.Println("        Unpinned features were auto-detected from source code analysis.")
		fmt.Println()
	}
}

// renderDoctorPerModuleProfiles shows each module's profile in multi-module workspaces.
func renderDoctorPerModuleProfiles(actx *analyzer.AnalysisContext) {
	if len(actx.FeatureProfiles) <= 1 {
		return
	}

	fmt.Println()
	fmt.Printf("PER-MODULE PROFILES (%d modules)\n", len(actx.FeatureProfiles))
	fmt.Println("─────────────────────────────────")
	fmt.Println()

	type modProfile struct {
		dir     string
		profile analyzer.FeatureProfile
	}

	mods := make([]modProfile, 0, len(actx.FeatureProfiles))
	for dir, p := range actx.FeatureProfiles {
		mods = append(mods, modProfile{dir, p})
	}

	sort.Slice(mods, func(i, j int) bool {
		return len(mods[i].dir) < len(mods[j].dir)
	})

	for _, m := range mods {
		rel := m.dir
		if rel == "" {
			rel = "(root)"
		}

		fmt.Printf("=== %s ===\n", rel)
		fmt.Print(m.profile)
		fmt.Println()
	}
}

// renderDoctorSuggestedConfig outputs a copy-pasteable .cqrs-lint.json features
// section based on the detected profile.
func renderDoctorSuggestedConfig(cfg *AppConfig, actx *analyzer.AnalysisContext) {
	profile := actx.FeatureProfile
	features := profile.ToConfigFeatures()

	raw, err := json.Marshal(
		map[string]analyzer.ConfigFeatures{"features": features},
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
	)
	if err != nil {
		return
	}

	fmt.Println("SUGGESTED .cqrs-lint.json")
	fmt.Println("─────────────────────────")
	fmt.Println()
	fmt.Println("  Copy-paste to pin the detected profile (prevents auto-detection drift):")
	fmt.Println()
	fmt.Println(string(raw))

	// Show rules overrides if loaded
	if len(cfg.Rules.ExternalAPIStructPrefixes) > 0 {
		rulesRaw, err := json.Marshal(
			map[string]analyzer.RulesConfig{"rules": cfg.Rules},
			jsontext.WithIndentPrefix(""),
			jsontext.WithIndent("  "),
		)
		if err == nil {
			fmt.Println()
			fmt.Println("  Loaded rules overrides:")
			fmt.Println()
			fmt.Println(string(rulesRaw))
		}
	}

	fmt.Println()
}

// renderDoctorSuppressions counts and displays inline //cqrs-lint:ignore comments.
func renderDoctorSuppressions(actx *analyzer.AnalysisContext) {
	suppressionCounts := countSuppressions(actx)
	if len(suppressionCounts) == 0 {
		return
	}

	fmt.Println("INLINE SUPPRESSIONS")
	fmt.Println("───────────────────")
	fmt.Println()
	fmt.Println("  //cqrs-lint:ignore(RULE) counts per rule:")
	fmt.Println()

	type suppressionEntry struct {
		rule  string
		count int
	}

	entries := make([]suppressionEntry, 0, len(suppressionCounts))
	for rule, count := range suppressionCounts {
		entries = append(entries, suppressionEntry{rule, count})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	for _, e := range entries {
		fmt.Printf("    %-8s  %d suppressed\n", e.rule, e.count)
	}

	fmt.Println()
	fmt.Println("  High suppression counts may signal a rule heuristic that needs tuning.")
	fmt.Println("  Consider reporting these as false positives.")
	fmt.Println()
}

// countSuppressions scans all Go files for //cqrs-lint:ignore(RULE) comments
// and returns a map of rule ID to suppression count.
func countSuppressions(actx *analyzer.AnalysisContext) map[string]int {
	counts := make(map[string]int)

	for _, gf := range actx.GoFiles {
		if gf.AST == nil {
			continue
		}

		for _, group := range gf.AST.Comments {
			for _, c := range group.List {
				text := c.Text
				idx := strings.Index(text, "cqrs-lint:ignore(")
				if idx < 0 {
					continue
				}

				start := idx + len("cqrs-lint:ignore(")
				end := strings.Index(text[start:], ")")
				if end < 0 {
					continue
				}

				rule := strings.TrimSpace(text[start : start+end])
				if rule != "" {
					counts[rule]++
				}
			}
		}
	}

	return counts
}
