package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
)

var errConfigExists = errors.New(".cqrs-lint.json already exists")

// initPresetFlags holds the --preset flag for the init command.
type initPresetFlags struct {
	Preset string `default:"" flag:"preset" help:"Config preset: local-cli, library, server, full-stack"`
}

// presetConfigs maps preset names to their generated .cqrs-lint.json content.
// Each preset tailors the feature profile and disabled rules for a common
// project type, eliminating the most common configuration churn.
//
//nolint:gochecknoglobals // static template table
var presetConfigs = map[string]string{
	"": `{
  "min-severity": "info",
  "min-confidence": "low",
  "format": "text",
  "exclude": [],
  "only": "",
  "features": {},
  "preset": ""
}
`,
	"local-cli": `{
  "min-severity": "warning",
  "min-confidence": "low",
  "format": "text",
  "features": {
    "server": false
  },
  "rules": {
    "disable": ["F004", "F009", "F013", "F015", "F017"]
  }
}
`,
	"library": `{
  "min-severity": "info",
  "min-confidence": "low",
  "format": "text",
  "features": {
    "server": false,
    "command-flow": "read-only"
  },
  "rules": {
    "disable": ["E003", "E016"]
  }
}
`,
	"server": `{
  "min-severity": "info",
  "min-confidence": "low",
  "format": "text",
  "features": {
    "server": true
  }
}
`,
	"full-stack": `{
  "min-severity": "info",
  "min-confidence": "low",
  "format": "text",
  "features": {}
}
`,
}

func setupInitCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, initPresetFlags](
		"init",
		initPresetFlags{},
		func(_ context.Context, _ *AppConfig, flags initPresetFlags) error {
			if _, err := os.Stat(".cqrs-lint.json"); err == nil {
				return errConfigExists
			}

			preset := strings.TrimSpace(flags.Preset)
			content, ok := presetConfigs[preset]
			if !ok {
				return fmt.Errorf(
					"unknown preset %q (available: local-cli, library, server, full-stack)",
					preset,
				)
			}

			if err := os.WriteFile(".cqrs-lint.json", []byte(content), 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			if preset == "" {
				fmt.Println("Created .cqrs-lint.json with default settings")
			} else {
				fmt.Printf("Created .cqrs-lint.json with preset %q\n", preset)
			}

			return nil
		},
		cmdguard.WithShort("Create a .cqrs-lint.json config file with defaults"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "init", cmd, err)
}
