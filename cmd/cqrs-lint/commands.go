package main

import (
	"context"
	"fmt"

	cmdguard "github.com/larsartmann/cmdguard/v3/pkg/cmdguard/v3"
)

// registerCommand wraps the create-and-add pattern shared by every subcommand:
// it surfaces NewCommand failures as "create <name> command" and AddCommand
// failures as "add <name> command".
func registerCommand[F any](
	cli *cmdguard.CLI[AppConfig],
	name string,
	cmd cmdguard.Command[AppConfig, F],
	err error,
) error {
	if err != nil {
		return fmt.Errorf("create %s command: %w", name, err)
	}
	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		return fmt.Errorf("add %s command: %w", name, err)
	}
	return nil
}

func setupRulesCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, cmdguard.NoFlags](
		"rules",
		cmdguard.NoFlags{},
		func(_ context.Context, cfg *AppConfig, _ cmdguard.NoFlags) error {
			out, err := renderRulesTable(parseColorMode(cfg.Color))
			if err != nil {
				return fmt.Errorf("render rules: %w", err)
			}

			fmt.Println(out)

			return nil
		},
		cmdguard.WithShort("List all available rules"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "rules", cmd, err)
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
	return registerCommand(cli, "version", cmd, err)
}
