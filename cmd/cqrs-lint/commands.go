package main

import (
	"context"
	"fmt"

	cmdguard "github.com/larsartmann/cmdguard/v3/pkg/cmdguard/v3"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

func setupRulesCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, cmdguard.NoFlags](
		"rules",
		cmdguard.NoFlags{},
		func(_ context.Context, _ *AppConfig, _ cmdguard.NoFlags) error {
			fmt.Print(rules.ListRules())

			return nil
		},
		cmdguard.WithShort("List all available rules"),
		cmdguard.WithNoArgs(),
	)
	if err != nil {
		return fmt.Errorf("create rules command: %w", err)
	}

	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		return fmt.Errorf("add rules command: %w", err)
	}

	return nil
}

func setupVersionCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, cmdguard.NoFlags](
		"version",
		cmdguard.NoFlags{},
		func(_ context.Context, _ *AppConfig, _ cmdguard.NoFlags) error {
			fmt.Printf("cqrs-lint %s\n", version)

			return nil
		},
		cmdguard.WithShort("Print version"),
		cmdguard.WithNoArgs(),
	)
	if err != nil {
		return fmt.Errorf("create version command: %w", err)
	}

	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		return fmt.Errorf("add version command: %w", err)
	}

	return nil
}
