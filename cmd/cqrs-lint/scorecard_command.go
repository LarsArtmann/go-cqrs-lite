package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// errScorecardBelowThreshold signals that the scorecard coverage is below the
// --scorecard-threshold gate. Returned so cmdguard sets a non-zero exit code.
var errScorecardBelowThreshold = errors.New("scorecard coverage below threshold")

// scorecardFlags adds --format, --color, and --scorecard-threshold to the scorecard subcommand.
type scorecardFlags struct {
	Format    string `default:"text" flag:"format"              help:"Output format (text, json, markdown)"            short:"o"`
	Color     string `default:"auto" flag:"color"               help:"Colored output: auto,always,never"`
	Threshold int    `default:"0"    flag:"scorecard-threshold" help:"Exit non-zero if coverage is below N% (CI gate)"`
}

func setupScorecardCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"scorecard",
		scorecardFlags{},
		func(_ context.Context, cfg *AppConfig, flags scorecardFlags) error {
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

			if flags.Threshold > 0 && result.Summary.CoveragePercent < flags.Threshold {
				fmt.Fprintf(os.Stderr,
					"scorecard coverage %d%% is below threshold %d%%\n",
					result.Summary.CoveragePercent, flags.Threshold)
				return fmt.Errorf("%w: %d%% < %d%%",
					errScorecardBelowThreshold,
					result.Summary.CoveragePercent, flags.Threshold)
			}

			return nil
		},
		cmdguard.WithShort("Show module adoption scorecard (used/missing/coverage)"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "scorecard", cmd, err)
}
