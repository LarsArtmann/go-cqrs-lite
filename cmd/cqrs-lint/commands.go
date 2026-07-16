package main

import (
	"context"
	"fmt"
	"os"

	cmdguard "github.com/larsartmann/cmdguard/v3/pkg/cmdguard/v3"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

func setupRulesCommand(cli *cmdguard.CLI[AppConfig]) {
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
		fmt.Fprintf(os.Stderr, "Error creating rules command: %v\n", err)
		os.Exit(1)
	}

	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding rules command: %v\n", err)
		os.Exit(1)
	}
}

func setupVersionCommand(cli *cmdguard.CLI[AppConfig]) {
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
		fmt.Fprintf(os.Stderr, "Error creating version command: %v\n", err)
		os.Exit(1)
	}

	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding version command: %v\n", err)
		os.Exit(1)
	}
}
