package main

import (
	"context"
	"encoding/json"
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

			features := profile.ToConfigFeatures()
			raw, err := json.MarshalIndent(
				map[string]analyzer.ConfigFeatures{"features": features},
				"",
				"  ",
			)
			if err != nil {
				return fmt.Errorf("marshal suggested features: %w", err)
			}

			fmt.Println("Suggested .cqrs-lint.json features section:")
			fmt.Println()
			fmt.Println(string(raw))

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
