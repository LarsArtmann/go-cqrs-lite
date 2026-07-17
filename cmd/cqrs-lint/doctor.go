package main

import (
	"context"
	"fmt"

	cmdguard "github.com/larsartmann/cmdguard/v3/pkg/cmdguard/v3"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func setupDoctorCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, cmdguard.NoFlags](
		"doctor",
		cmdguard.NoFlags{},
		func(ctx context.Context, cfg *AppConfig, _ cmdguard.NoFlags) error {
			actx, err := analyzer.BuildContext(cfg.Path)
			if err != nil {
				return fmt.Errorf("load packages: %w", err)
			}

			profile := actx.FeatureProfile

			fmt.Println("Detected go-cqrs-lite feature profile:")
			fmt.Println()
			fmt.Print(profile)
			fmt.Println()

			fmt.Println("Suggested .cqrs-lint.json features section:")
			fmt.Println()
			fmt.Printf("  \"features\": {\n")
			if profile.Store != analyzer.StoreUnknown && profile.Store != analyzer.StoreNone {
				fmt.Printf("    \"store\": \"%s\",\n", profile.Store)
			}
			if profile.CommandFlow != analyzer.CommandFlowUnknown {
				fmt.Printf("    \"command-flow\": \"%s\",\n", profile.CommandFlow)
			}
			fmt.Printf("    \"server\": %t,\n", profile.HasServer)
			fmt.Printf("    \"soft-delete\": %t,\n", profile.HasSoftDelete)
			if profile.Tracing != analyzer.TracingUnknown {
				fmt.Printf("    \"tracing\": \"%s\",\n", profile.Tracing)
			}
			if profile.Snapshot != analyzer.SnapshotUnknown {
				fmt.Printf("    \"snapshot\": \"%s\"\n", profile.Snapshot)
			}
			fmt.Printf("  }\n")

			return nil
		},
		cmdguard.WithShort("Detect and display the project's go-cqrs-lite feature profile"),
		cmdguard.WithNoArgs(),
	)
	if err != nil {
		return fmt.Errorf("create doctor command: %w", err)
	}

	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		return fmt.Errorf("add doctor command: %w", err)
	}

	return nil
}
