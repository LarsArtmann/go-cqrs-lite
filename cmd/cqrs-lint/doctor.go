package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
)

// doctorFlags adds subcommand-level flags to the doctor command.
type doctorFlags struct {
	AuditSuppressions bool `default:"false" flag:"audit-suppressions" help:"Audit all inline suppressions: show active vs stale vs unknown-rule status"`
	Fix               bool `default:"false" flag:"fix"                help:"Remove stale whole-line suppressions (implies audit)"`
}

func setupDoctorCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"doctor",
		doctorFlags{},
		func(ctx context.Context, cfg *AppConfig, flags doctorFlags) error {
			actx, err := analyzer.BuildContext(cfg.Path)
			if err != nil {
				return fmt.Errorf("load packages: %w", err)
			}

			if flags.AuditSuppressions || flags.Fix {
				return runSuppressionAudit(ctx, cfg, actx, flags.Fix)
			}

			renderDoctorLoadErrors(os.Stderr, actx)
			renderDoctorConfigFile(os.Stdout, cfg)

			// Resolve all overrides so the rest of the output shows the
			// EFFECTIVE configuration, not just the raw detected values.
			applyConfigOverrides(cfg, actx)

			renderDoctorPreset(os.Stdout, cfg)
			renderDoctorEffectiveSettings(os.Stdout, cfg)
			renderDoctorFeatureProfile(os.Stdout, actx)
			renderDoctorPerModuleProfiles(os.Stdout, actx)
			renderDoctorSuggestedConfig(os.Stdout, cfg, actx)
			renderDoctorSuppressions(os.Stdout, actx)

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
func renderDoctorLoadErrors(w io.Writer, actx *analyzer.AnalysisContext) {
	if len(actx.LoadErrors) == 0 {
		return
	}

	_, _ = fmt.Fprintln(
		w,
		"WARNING: package loading was partial; the profile below may be incomplete or misleading.",
	)
	for _, le := range actx.LoadErrors {
		if le.PkgPath != "" {
			_, _ = fmt.Fprintf(w, "  %s (%s):\n", le.Module, le.PkgPath)
		} else {
			_, _ = fmt.Fprintf(w, "  %s:\n", le.Module)
		}
		for _, msg := range le.Errors {
			_, _ = fmt.Fprintf(w, "    %s\n", msg)
		}
	}
	_, _ = fmt.Fprintln(w)
}

// renderDoctorConfigFile shows the raw .cqrs-lint.json content (if present),
// including parent configs found in ancestor directories.
func renderDoctorConfigFile(w io.Writer, cfg *AppConfig) {
	_, _ = fmt.Fprintln(w, "CONFIG FILE")
	_, _ = fmt.Fprintln(w, "───────────")

	configPath := filepath.Join(cfg.Path, ".cqrs-lint.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		absPath, _ := filepath.Abs(configPath)
		_, _ = fmt.Fprintf(w, "  Path:    %s\n", absPath)
		_, _ = fmt.Fprintln(w, "  Status:  NOT FOUND")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "  No .cqrs-lint.json found. Run 'cqrs-lint init' to create one,")
		_, _ = fmt.Fprintln(w, "  or 'cqrs-lint explain' to see all available options.")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w)
		return
	}

	absPath, _ := filepath.Abs(configPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	_, _ = fmt.Fprintf(w, "  Path:    %s\n", absPath)
	_, _ = fmt.Fprintf(w, "  Status:  Found (%d lines, %d bytes)\n", len(lines), len(data))
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "  Content:")

	for _, line := range lines {
		_, _ = fmt.Fprintf(w, "    %s\n", line)
	}

	// Show parent configs (monorepo inheritance)
	parents := findParentConfigs(cfg.Path)
	if len(parents) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "  Parent configs (merged for rule disables):")
		for _, p := range parents {
			rel, err := filepath.Rel(cfg.Path, p)
			if err != nil {
				rel = p
			}
			_, _ = fmt.Fprintf(w, "    %s\n", rel)
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w)
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
func renderDoctorPreset(w io.Writer, cfg *AppConfig) {
	_, _ = fmt.Fprintln(w, "ACTIVE PRESET")
	_, _ = fmt.Fprintln(w, "─────────────")

	if cfg.Preset == "" {
		_, _ = fmt.Fprintln(w, "  (none)")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "  No preset active. All features are auto-detected.")
		_, _ = fmt.Fprintln(w, "  See 'cqrs-lint explain' for available presets.")
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w)
		return
	}

	desc := presetDescriptions[cfg.Preset]
	presetDef := analyzer.ResolvePresetDefinition(cfg.Preset)

	_, _ = fmt.Fprintf(w, "  %s\n", cfg.Preset)
	if desc != "" {
		_, _ = fmt.Fprintf(w, "    %s\n", desc)
	}

	if f := formatConfigFeatures(presetDef.Features); f != "" {
		_, _ = fmt.Fprintf(w, "    Features pinned:  %s\n", f)
	}

	if len(presetDef.Rules.Disable) > 0 {
		_, _ = fmt.Fprintf(
			w,
			"    Rules disabled:   %s\n",
			strings.Join(presetDef.Rules.Disable, ", "),
		)
	}

	if presetDef.MinSeverity != "" {
		_, _ = fmt.Fprintf(w, "    Severity floor:   %s\n", presetDef.MinSeverity)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w)
}

// renderDoctorEffectiveSettings shows the resolved configuration after all
// overrides (preset, config file, auto-detection) are applied.
func renderDoctorEffectiveSettings(w io.Writer, cfg *AppConfig) {
	_, _ = fmt.Fprintln(w, "EFFECTIVE SETTINGS")
	_, _ = fmt.Fprintln(w, "──────────────────")
	_, _ = fmt.Fprintln(w)

	presetDef := analyzer.ResolvePresetDefinition(cfg.Preset)

	effectiveSev := resolveMinSeverity(presetDef.MinSeverity, cfg.MinSeverity)
	sevSource := "default"
	if presetDef.MinSeverity != "" && effectiveSev == presetDef.MinSeverity {
		sevSource = "preset floor"
	} else if effectiveSev != "info" {
		sevSource = "config"
	}

	_, _ = fmt.Fprintf(w, "  min-severity:    %s  (%s)\n", effectiveSev, sevSource)
	_, _ = fmt.Fprintf(w, "  min-confidence:  %s\n", cfg.MinConfidence)
	_, _ = fmt.Fprintf(w, "  format:          %s\n", cfg.Format)
	_, _ = fmt.Fprintf(w, "  color:           %s\n", cfg.Color)

	if cfg.Preset != "" {
		_, _ = fmt.Fprintf(w, "  preset:          %s\n", cfg.Preset)
	}

	// Disabled rules with source breakdown
	totalRules := len(rules.AllRules())
	disabledCount := len(cfg.Rules.Disable)
	activeCount := totalRules - disabledCount
	_, _ = fmt.Fprintf(w, "  rules:           %d total, %d active, %d disabled\n",
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
			_, _ = fmt.Fprintf(w, "    from preset:   %s\n", strings.Join(fromPreset, ", "))
		}
		if len(fromConfig) > 0 {
			_, _ = fmt.Fprintf(w, "    from config:   %s\n", strings.Join(fromConfig, ", "))
		}
	}

	// Rules overrides
	if len(cfg.Rules.ExternalAPIStructPrefixes) > 0 {
		_, _ = fmt.Fprintf(w, "  external-api-struct-prefixes: %s\n",
			strings.Join(cfg.Rules.ExternalAPIStructPrefixes, ", "))
	}
	if len(cfg.Rules.IgnoreFloatFields) > 0 {
		_, _ = fmt.Fprintf(w, "  c008-ignore-fields: %s\n",
			strings.Join(cfg.Rules.IgnoreFloatFields, ", "))
	}
	if len(cfg.Rules.IgnoreStructs) > 0 {
		_, _ = fmt.Fprintf(w, "  c008-ignore-structs: %s\n",
			strings.Join(cfg.Rules.IgnoreStructs, ", "))
	}

	// Health config
	if cfg.Health.InfoCap != 0 {
		_, _ = fmt.Fprintf(w, "  health.info-cap: %d\n", cfg.Health.InfoCap)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w)
}

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
		cf.Domain != nil || cf.Transport != nil || cf.ServerLocal != nil || cf.AsyncBus != nil

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
		return len(mods[i].dir) < len(mods[j].dir)
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
