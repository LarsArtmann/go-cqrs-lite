package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/fix"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/suppression"
)

// errAbortClean signals that the lint found nothing to analyze but no error
// occurred — the run should exit zero (e.g. no Go files, no CQRS imports).
var errAbortClean = errors.New("nothing to lint")

func run(ctx context.Context, cfg *AppConfig) error {
	start := time.Now()

	actx, err := analyzer.BuildContext(cfg.Path)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	applyConfigOverrides(cfg, actx)

	if err := handleLoadErrors(cfg, actx); err != nil {
		if errors.Is(err, errAbortClean) {
			return nil
		}
		return err
	}

	detectors := selectDetectors(cfg, actx)

	result, err := runPipeline(ctx, cfg, detectors)
	if err != nil {
		return err
	}

	active, unsuppressed, suppressed := filterFindings(cfg, collectFindings(result))

	printSummary(cfg, actx, start, active, unsuppressed, len(suppressed), detectors, result)

	if err := outputFindings(ctx, active, cfg); err != nil {
		return fmt.Errorf("output: %w", err)
	}

	if cfg.ShowSuppressed && len(suppressed) > 0 && !cfg.Quiet {
		formatSuppressedFindings(os.Stdout, suppressed, parseColorMode(cfg.Color))
	}

	if cfg.HealthScore {
		infoCap := cfg.Health.InfoCap
		if infoCap == 0 {
			infoCap = defaultInfoDeductionCap
		}
		hs := ComputeHealthScoreWithCap(unsuppressed, infoCap)
		fmt.Print(renderHealthScore(hs, parseColorMode(cfg.Color)))
	}

	return shouldExitWithError(cfg, active)
}

// applyConfigOverrides merges config-declared feature overrides and rule-specific
// overrides onto the auto-detected analysis context. Also validates rule config
// keys to surface typos that would silently disable an override.
func applyConfigOverrides(cfg *AppConfig, actx *analyzer.AnalysisContext) {
	actx.FeatureProfile = analyzer.ResolveFeatureProfile(
		cfg.Features,
		cfg.Preset,
		actx.FeatureProfile,
	)

	rawRules := loadRawRulesJSON()
	cfg.Rules.Validate(os.Stderr, rawRules)
	actx.RulesConfig = cfg.Rules
}

// handleLoadErrors processes package-loading failures and returns an error
// if the lint should abort (--strict-load, no files with errors). Returns nil
// when the analysis can proceed, possibly after printing a partial-analysis warning.
func handleLoadErrors(cfg *AppConfig, actx *analyzer.AnalysisContext) error {
	if cfg.StrictLoad && len(actx.LoadErrors) > 0 {
		fmt.Fprintln(os.Stderr, "cqrs-lint: --strict-load mode — package loading reported errors:")
		fmt.Fprintln(os.Stderr)
		printLoadErrors(os.Stderr, actx.LoadErrors)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(
			os.Stderr,
			"Analyzed %d file(s) with %d load error(s). --strict-load requires zero load errors.\n",
			len(actx.GoFiles),
			len(actx.LoadErrors),
		)
		return errFindingsWithErrors
	}

	if len(actx.GoFiles) == 0 {
		if len(actx.LoadErrors) > 0 {
			fmt.Fprintln(os.Stderr, "cqrs-lint: could not analyze any packages.")
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(
				os.Stderr,
				"This usually means the project does not compile. Package loading reported errors:",
			)
			fmt.Fprintln(os.Stderr)
			printLoadErrors(os.Stderr, actx.LoadErrors)
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(
				os.Stderr,
				"Fix the build errors above (try `go build ./...`), then re-run cqrs-lint.",
			)
			fmt.Fprintln(os.Stderr, "Nothing was analyzed; this is not a clean bill of health.")
			return errFindingsWithErrors
		}

		if !cfg.Quiet {
			if len(actx.Packages) > 0 {
				fmt.Fprintln(
					os.Stderr,
					"Found Go files but none import go-cqrs-lite. Nothing to lint.",
				)
			} else {
				fmt.Fprintln(os.Stderr, "No Go files found. Nothing to lint.")
			}
		}

		return errAbortClean
	}

	if !cfg.Quiet && len(actx.LoadErrors) > 0 {
		fmt.Fprintf(
			os.Stderr,
			"WARNING: %d package(s) failed to load; analysis is partial.\n",
			len(actx.LoadErrors),
		)
		fmt.Fprintln(os.Stderr, "Use --verbose for details or --strict to fail on any load error.")
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

// selectDetectors builds the detector list based on mode (fast, all, filtered).
func selectDetectors(cfg *AppConfig, actx *analyzer.AnalysisContext) []finding.Detector {
	if cfg.FastMode {
		return rules.RegisterCritical(actx)
	}

	detectors := rules.RegisterAll(actx)

	if cfg.Categories != "" {
		parts := strings.Split(cfg.Categories, ",")
		if rules.IsRuleID(parts[0]) {
			return rules.FilterByRuleIDs(detectors, parts)
		}

		return rules.FilterByCategory(detectors, parts)
	}

	return detectors
}

// runPipeline builds the pipeline configuration, creates the pipeline, and runs it.
func runPipeline(
	ctx context.Context,
	cfg *AppConfig,
	detectors []finding.Detector,
) (*pipeline.PipelineResult, error) {
	pipeConfig := pipeline.Config{
		MaxIterations:       5,
		ParallelDetectors:   true,
		GracefulDegradation: true,
		DryRun:              !cfg.Fix,
		Timeout:             5 * time.Minute,
		Processors: []pipeline.FindingTransformer{
			suppression.NewSuppressionFilter(),
		},
	}

	if cfg.Fix || cfg.DryRun {
		pipeConfig.FixProviders = []pipeline.FixProvider{fix.NewCQRSFixProvider()}
	}

	pipe, err := pipeline.New(pipeConfig, cfg.Path, detectors...)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}

	result, err := pipe.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline run: %w", err)
	}

	return result, nil
}

// filterFindings applies path exclusion, suppression splitting, severity, and
// confidence filters. Returns active findings (for output + exit), unsuppressed
// findings (for health score + stale suppression detection), and suppressed
// findings (for --show-suppressed auditing).
func filterFindings(
	cfg *AppConfig,
	allFindings []finding.Finding,
) (active, unsuppressed, suppressed []finding.Finding) {
	if cfg.Exclude != "" {
		allFindings = filterByExcludedPaths(allFindings, strings.Split(cfg.Exclude, ","))
	}

	unsuppressed, suppressed = filterSuppressed(allFindings)

	active = filterBySeverity(unsuppressed, cfg.MinSeverity)
	if cfg.FPSuspects {
		active = filterFPSuspects(active)
	} else {
		active = filterByConfidence(active, cfg.MinConfidence)
	}

	return active, unsuppressed, suppressed
}

// printSummary writes the text-mode analysis summary to stderr, including
// timing, suppression counts, stale-suppression warnings, and verbose detail.
func printSummary(
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
	start time.Time,
	active, unsuppressed []finding.Finding,
	suppressedCount int,
	detectors []finding.Detector,
	result *pipeline.PipelineResult,
) {
	if !cfg.Quiet && cfg.Format == "text" {
		elapsed := time.Since(start)
		fmt.Fprintf(
			os.Stderr,
			"Analyzed %d files in %s\n",
			len(actx.GoFiles),
			elapsed.Round(time.Millisecond),
		)
		if suppressedCount > 0 {
			fmt.Fprintf(os.Stderr, "%d finding(s) suppressed by inline comments\n", suppressedCount)
		}

		goFilePaths := make([]string, 0, len(actx.GoFiles))
		for _, gf := range actx.GoFiles {
			goFilePaths = append(goFilePaths, gf.Path)
		}

		stale := suppression.DetectStaleSuppressions(goFilePaths, unsuppressed)
		for _, s := range stale {
			fmt.Fprintln(os.Stderr, suppression.FormatStaleWarning(s))
		}

		if cfg.FPSuspects {
			fmt.Fprintf(os.Stderr,
				"Showing %d low-confidence finding(s) — likely false positives.\n"+
					"Suppress confirmed FPs with //cqrs-lint:ignore(RULE)\n",
				len(active))
		}

		fmt.Fprintln(os.Stderr)
	}

	if cfg.Verbose && !cfg.Quiet {
		modules := countModules(actx.GoFiles)
		fmt.Fprintf(os.Stderr, "Modules: %d  Detectors: %d  Findings: %d (before filtering)\n\n",
			modules, len(detectors), len(unsuppressed)+suppressedCount)
		fmt.Fprintf(os.Stderr, "Feature profile:\n%s\n", actx.FeatureProfile.String())
		if len(actx.LoadErrors) > 0 {
			fmt.Fprintf(os.Stderr, "Load errors (%d):\n", len(actx.LoadErrors))
			printLoadErrors(os.Stderr, actx.LoadErrors)
			fmt.Fprintln(os.Stderr)
		}
		printDetectorTimings(os.Stderr, result.Metrics)
	}
}
