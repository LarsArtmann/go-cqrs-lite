package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

var errConfigExists = errors.New(".cqrs-lint.json already exists")

// initPresetFlags holds the --preset flag for the init command.
type initPresetFlags struct {
	Preset string `default:"" flag:"preset" help:"Config preset: local-cli, production, library, library-framework, read-only, v5-ready"`
}

func setupInitCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"init",
		initPresetFlags{},
		func(_ context.Context, cfg *AppConfig, flags initPresetFlags) error {
			// --path is honored (the root command's default "."): the config
			// lands in the directory the user pointed at, not silently in
			// the CWD. A missing directory is an error, not a mkdir.
			if _, err := os.Stat(cfg.Path); err != nil {
				return fmt.Errorf("--path %s: %w", cfg.Path, err)
			}

			target := filepath.Join(cfg.Path, ".cqrs-lint.json")
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("%w: %s", errConfigExists, target)
			}

			preset := strings.TrimSpace(flags.Preset)

			content, err := generateInitConfig(preset)
			if err != nil {
				return err
			}

			if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			if preset == "" {
				fmt.Printf("Created %s with default settings\n", target)
				fmt.Println("Run 'cqrs-lint explain' for full documentation of all config keys.")
			} else {
				fmt.Printf("Created %s with preset %q\n", target, preset)
				fmt.Println("Run 'cqrs-lint doctor' to see the resolved feature profile.")
				fmt.Println("Run 'cqrs-lint explain' for full documentation of all config keys.")
			}

			return nil
		},
		cmdguard.WithShort("Create a .cqrs-lint.json config file with defaults"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "init", cmd, err)
}

// generateInitConfig produces the .cqrs-lint.json content for the given preset.
// The output includes JSONC comments (// lines) explaining each setting, so
// users immediately see that comments are supported and understand what each
// key does. Returns an error for unknown preset names.
func generateInitConfig(preset string) (string, error) {
	if preset == "" {
		return defaultConfigTemplate(), nil
	}

	if !analyzer.IsKnownPreset(analyzer.ConfigPreset(preset)) {
		//cqrs-lint:ignore(C025) no underlying error to wrap — this is a new validation error
		return "", fmt.Errorf( //nolint:err113 // preset name is dynamic
			"unknown preset %q (available: %s)",
			preset,
			strings.Join(analyzer.ValidPresetNames(), ", "),
		)
	}

	return presetConfigTemplate(preset), nil
}

// defaultConfigTemplate returns a commented default config with all core knobs.
func defaultConfigTemplate() string {
	return `{
  // cqrs-lint configuration — JSON with Comments is supported
  // Run 'cqrs-lint explain' for full documentation of all keys

  // Minimum severity to show: info, warning, error, critical
  "min-severity": "info",

  // Minimum confidence to show: low, medium, high
  "min-confidence": "low",

  // Output format: ` + formatList(formatsLint) + `
  "format": "text",

  // Group findings by: none, module, aggregate
  // Uncomment to enable:
  // "group-by": "module"

  // Scorecard waivers: record WHY an adoptable module has no place in this
  // domain. A waiver renders in a WAIVED section (never silently hides),
  // leaves the coverage denominator, and should carry a revisit trigger.
  // Uncomment and edit per module:
  // "scorecard": {
  //   "waivers": [
  //     {
  //       "key": "graph",
  //       "reason": "no traversal-heavy read models",
  //       "trigger": "variable-depth queries appear"
  //     }
  //   ]
  // }
}
`
}

// presetConfigTemplate returns a commented config for a named preset.
func presetConfigTemplate(preset string) string {
	desc := presetDescriptions[analyzer.ConfigPreset(preset)]
	presetDef := analyzer.ResolvePresetDefinition(analyzer.ConfigPreset(preset))

	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("  // ")
	b.WriteString(desc)
	b.WriteString("\n  // Run 'cqrs-lint explain' for full documentation\n")
	fmt.Fprintf(&b, "  \"preset\": \"%s\"", preset)

	if presetDef.MinSeverity != "" {
		b.WriteString(",\n\n")
		b.WriteString("  // Minimum severity: \"")
		b.WriteString(presetDef.MinSeverity)
		b.WriteString("\" is the preset floor (lower bound).\n")
		b.WriteString("  // You can raise this (e.g. to \"error\") but not lower it.\n")
		fmt.Fprintf(&b, "  \"min-severity\": \"%s\"", presetDef.MinSeverity)
	}

	if len(presetDef.Rules.SeverityOverrides) > 0 {
		b.WriteString(",\n\n")
		b.WriteString("  // Severity overrides: rewrite a rule's catalog severity\n")
		b.WriteString("  // (explicit config entries win over the preset's).\n")
		b.WriteString("  \"rules\": {\n    \"severity-overrides\": {")

		ids := make([]string, 0, len(presetDef.Rules.SeverityOverrides))
		for id := range presetDef.Rules.SeverityOverrides {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		for i, id := range ids {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%q: %q", id, presetDef.Rules.SeverityOverrides[id])
		}
		b.WriteString("}\n  }")
	}

	b.WriteString("\n}\n")
	return b.String()
}
