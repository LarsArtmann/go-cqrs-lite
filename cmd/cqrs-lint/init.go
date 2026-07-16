package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	cmdguard "github.com/larsartmann/cmdguard/v3/pkg/cmdguard/v3"
)

var errConfigExists = errors.New(".cqrs-lint.json already exists")

const configTemplate = `{
  "min_severity": "info",
  "min_confidence": "low",
  "format": "text",
  "exclude": [],
  "only": ""
}
`

func setupInitCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, cmdguard.NoFlags](
		"init",
		cmdguard.NoFlags{},
		func(_ context.Context, _ *AppConfig, _ cmdguard.NoFlags) error {
			if _, err := os.Stat(".cqrs-lint.json"); err == nil {
				return errConfigExists
			}

			if err := os.WriteFile(".cqrs-lint.json", []byte(configTemplate), 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			fmt.Println("Created .cqrs-lint.json with default settings")

			return nil
		},
		cmdguard.WithShort("Create a .cqrs-lint.json config file with defaults"),
		cmdguard.WithNoArgs(),
	)
	if err != nil {
		return fmt.Errorf("create init command: %w", err)
	}

	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		return fmt.Errorf("add init command: %w", err)
	}

	return nil
}
