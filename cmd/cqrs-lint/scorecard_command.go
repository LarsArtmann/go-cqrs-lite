package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// errScorecardBelowThreshold signals that the scorecard coverage is below the
// --scorecard-threshold gate. Returned so cmdguard sets a non-zero exit code.
var errScorecardBelowThreshold = errors.New("scorecard coverage below threshold")

// scorecardFlags adds --scorecard-threshold to the scorecard subcommand.
// --format/-o and --color are inherited from the root command (and the config
// file) so every command honors the SAME output knobs — no per-subcommand
// shadow copies with divergent defaults.
type scorecardFlags struct {
	Threshold int `default:"0" flag:"scorecard-threshold" help:"Exit non-zero if coverage is below N% (CI gate)"`
}

func setupScorecardCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand(
		"scorecard",
		scorecardFlags{},
		func(ctx context.Context, cfg *AppConfig, flags scorecardFlags) error {
			if err := validateFormatFlag(cfg.Format, formatsScorecard...); err != nil {
				return err
			}

			actx, err := analyzer.BuildContext(cfg.Path)
			if err != nil {
				return fmt.Errorf("load packages: %w", err)
			}

			if err := resolveScorecardPreset(cfg); err != nil {
				return err
			}
			applyConfigOverrides(cfg, actx)

			return runScorecard(ctx, cfg, actx, flags.Threshold)
		},
		cmdguard.WithShort(
			withFormatsSuffix(
				"Show module adoption scorecard (used/missing/coverage)",
				formatsScorecard,
			),
		),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "scorecard", cmd, err)
}

// resolveScorecardPreset fixes the effective preset to the scored project's
// own config before feature overrides resolve: a preset recorded in
// <path>/.cqrs-lint.json wins, and the operator's cwd config (cmdguard
// loader) fills only when the project records none. Without this,
// `scorecard --path X` run from a config-carrying cwd scores X against the
// operator's preset instead of its own. The waivers follow the same
// convention (resolveScorecardWaivers).
func resolveScorecardPreset(cfg *AppConfig) error {
	project, found, err := analyzer.LoadProjectConfig(cfg.Path)
	if err != nil {
		return fmt.Errorf("scorecard: %w", err)
	}
	if found && project.Preset != "" && project.Preset != analyzer.PresetNone {
		cfg.Preset = project.Preset
	}
	return nil
}

// runScorecard is the single scorecard entry point shared by the root
// --scorecard flag and the scorecard subcommand. Both compute the same result
// (including the deprecated-modules panel) and render with the shared
// --format/--color. Only the subcommand passes a non-zero threshold to arm
// the CI coverage gate.
func runScorecard(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
	threshold int,
) error {
	usage := analyzer.DetectUsedModules(actx.Packages, actx.GoFiles, analyzer.DefaultCatalog)
	waivers, err := resolveScorecardWaivers(cfg)
	if err != nil {
		return fmt.Errorf("scorecard: %w", err)
	}
	result, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, usage, actx.FeatureProfile, cfg.Preset, waivers,
	)
	if err != nil {
		return fmt.Errorf("scorecard: %w", err)
	}
	result.Deprecated = ComputeDeprecatedPanel(ctx, actx)
	result.Summary.ModernityGrade = ModernityGrade(result.Deprecated, actx.FeatureProfile)

	out, err := renderScorecard(result, cfg.Format, parseColorMode(cfg.Color))
	if err != nil {
		return fmt.Errorf("render scorecard: %w", err)
	}

	fmt.Print(out)

	if threshold > 0 && result.Summary.CoveragePercent < threshold {
		fmt.Fprintf(os.Stderr,
			"scorecard coverage %d%% is below threshold %d%%\n",
			result.Summary.CoveragePercent, threshold)
		return fmt.Errorf("%w: %d%% < %d%%",
			errScorecardBelowThreshold,
			result.Summary.CoveragePercent, threshold)
	}

	return nil
}
