package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
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

// defaultConfigSkeleton is the minimal set of commonly-tuned knobs, written at
// their default values so new users can see what's available without noise.
type defaultConfigSkeleton struct {
	MinSeverity   string `json:"min-severity"`   //nolint:tagliatelle // CLI config key
	MinConfidence string `json:"min-confidence"` //nolint:tagliatelle // CLI config key
	Format        string `json:"format"`
}

// presetConfigSkeleton writes just the preset name — the runtime resolves
// features and rule defaults from PresetDefinitions at lint time. This is DRY:
// changing a preset's behaviour only requires updating PresetDefinitions.
// MinSeverity is included when the preset recommends a non-default floor
// (e.g. local-cli recommends "warning"), so the user sees and can edit it.
type presetConfigSkeleton struct {
	Preset      string `json:"preset"`
	MinSeverity string `json:"min-severity,omitempty"` //nolint:tagliatelle // CLI config key
}

// generateInitConfig produces the .cqrs-lint.json content for the given preset.
// For the empty default it writes a clean skeleton with core knobs at defaults.
// For a named preset it writes just the preset name (the runtime resolves the
// rest). Returns an error for unknown preset names.
func generateInitConfig(preset string) (string, error) {
	if preset == "" {
		return marshalInitConfig(defaultConfigSkeleton{
			MinSeverity:   "info",
			MinConfidence: "low",
			Format:        "text",
		})
	}

	if !analyzer.IsKnownPreset(analyzer.ConfigPreset(preset)) {
		return "", fmt.Errorf( //nolint:err113 // preset name is dynamic
			"unknown preset %q (available: %s)",
			preset,
			strings.Join(analyzer.ValidPresetNames(), ", "),
		)
	}

	return marshalInitConfig(presetConfigSkeleton{
		Preset:      preset,
		MinSeverity: analyzer.ResolvePresetDefinition(analyzer.ConfigPreset(preset)).MinSeverity,
	})
}

// marshalInitConfig serializes v to indented JSON with a trailing newline,
// matching the formatting of hand-written config files.
func marshalInitConfig(v any) (string, error) {
	raw, err := json.Marshal(v,
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
	)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	return string(raw) + "\n", nil
}
