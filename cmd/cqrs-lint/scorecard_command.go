package main

import (
	"context"
	"fmt"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// scorecardFlags adds --format and --color to the scorecard subcommand.
type scorecardFlags struct {
	Format string `default:"text" flag:"format" help:"Output format (text, json)"        short:"o"`
	Color  string `default:"auto" flag:"color"  help:"Colored output: auto,always,never"`
}

func setupScorecardCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"scorecard",
		scorecardFlags{},
		func(ctx context.Context, cfg *AppConfig, flags scorecardFlags) error {
			actx, err := analyzer.BuildContext(cfg.Path)
			if err != nil {
				return fmt.Errorf("load packages: %w", err)
			}

			applyConfigOverrides(cfg, actx)

			usage := analyzer.DetectUsedModules(
				actx.Packages,
				actx.GoFiles,
				analyzer.DefaultCatalog,
			)
			result := ComputeScorecard(
				analyzer.DefaultCatalog, usage,
				actx.FeatureProfile, cfg.Preset,
			)

			out, err := renderScorecard(result, flags.Format, parseColorMode(flags.Color))
			if err != nil {
				return fmt.Errorf("render scorecard: %w", err)
			}

			fmt.Print(out)
			return nil
		},
		cmdguard.WithShort("Show module adoption scorecard (used/missing/coverage)"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "scorecard", cmd, err)
}
