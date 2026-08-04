package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

var errConfigExists = errors.New(".cqrs-lint.json already exists")

// initPresetFlags holds the --preset flag for the init command.
type initPresetFlags struct {
	Preset string `default:"" flag:"preset" help:"Config preset: local-cli, production, library, read-only"`
}

func setupInitCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"init",
		initPresetFlags{},
		func(_ context.Context, _ *AppConfig, flags initPresetFlags) error {
			if _, err := os.Stat(".cqrs-lint.json"); err == nil {
				return errConfigExists
			}

			preset := strings.TrimSpace(flags.Preset)

			content, err := generateInitConfig(preset)
			if err != nil {
				return err
			}

			if err := os.WriteFile(".cqrs-lint.json", []byte(content), 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			if preset == "" {
				fmt.Println("Created .cqrs-lint.json with default settings")
			} else {
				fmt.Printf("Created .cqrs-lint.json with preset %q\n", preset)
				fmt.Println("Run 'cqrs-lint doctor' to see the resolved feature profile.")
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

  // Output format: text, json, sarif, markdown
  "format": "text"
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

	b.WriteString("\n}\n")
	return b.String()
}
