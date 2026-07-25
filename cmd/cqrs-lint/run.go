package main

import (
	"context"
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

func run(ctx context.Context, cfg *AppConfig) error {
	start := time.Now()

	actx, err := analyzer.BuildContext(cfg.Path)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	// Apply config-declared feature overrides on top of auto-detection.
	actx.FeatureProfile = analyzer.ResolveFeatureProfile(
		cfg.Features,
		cfg.Preset,
		actx.FeatureProfile,
	)

	// Rule-specific overrides (external-API allowlists, etc.).
	actx.RulesConfig = cfg.Rules

	// Validate + normalize rule overrides: warn on typos, unknown keys, and
	// empty/duplicate prefixes. Closes the silent-failure gap where a typo'd
	// config key (e.g. "external-api-prefixes") silently disabled an override.
	rawRules := loadRawRulesJSON()
	cfg.Rules.Validate(os.Stderr, rawRules)
	actx.RulesConfig = cfg.Rules

	// --strict-load: any package load error is fatal, even if some packages loaded.
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

		return nil
	}

	// Warn about partial analysis (non-strict mode with load errors).
	if !cfg.Quiet && len(actx.LoadErrors) > 0 {
		fmt.Fprintf(
			os.Stderr,
			"WARNING: %d package(s) failed to load; analysis is partial.\n",
			len(actx.LoadErrors),
		)
		fmt.Fprintln(os.Stderr, "Use --verbose for details or --strict to fail on any load error.")
		fmt.Fprintln(os.Stderr)
	}

	var detectors []finding.Detector
	if cfg.FastMode {
		detectors = rules.RegisterCritical(actx)
	} else {
		detectors = rules.RegisterAll(actx)
		if cfg.Categories != "" {
			parts := strings.Split(cfg.Categories, ",")
			if rules.IsRuleID(parts[0]) {
				detectors = rules.FilterByRuleIDs(detectors, parts)
			} else {
				detectors = rules.FilterByCategory(detectors, parts)
			}
		}
	}

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
		return fmt.Errorf("create pipeline: %w", err)
	}

	result, err := pipe.Run(ctx)
	if err != nil {
		return fmt.Errorf("pipeline run: %w", err)
	}

	allFindings := collectFindings(result)
	if cfg.Exclude != "" {
		allFindings = filterByExcludedPaths(allFindings, strings.Split(cfg.Exclude, ","))
	}

	// Split suppressed findings (//cqrs-lint:ignore) from active ones.
	// unsuppressedFindings feeds the severity/confidence filters and health
	// score. suppressedFindings is retained for --show-suppressed auditing.
	unsuppressedFindings, suppressedFindings := filterSuppressed(allFindings)
	suppressedCount := len(suppressedFindings)

	activeFindings := filterBySeverity(unsuppressedFindings, cfg.MinSeverity)
	if cfg.FPSuspects {
		// --fp-suspects: show only low-confidence findings (likely false
		// positives). Overrides the normal confidence filter.
		activeFindings = filterFPSuspects(activeFindings)
	} else {
		activeFindings = filterByConfidence(activeFindings, cfg.MinConfidence)
	}

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

		stale := suppression.DetectStaleSuppressions(goFilePaths, unsuppressedFindings)
		for _, s := range stale {
			fmt.Fprintln(os.Stderr, suppression.FormatStaleWarning(s))
		}

		if cfg.FPSuspects {
			fmt.Fprintf(os.Stderr,
				"Showing %d low-confidence finding(s) — likely false positives.\n"+
					"Suppress confirmed FPs with //cqrs-lint:ignore(RULE)\n",
				len(activeFindings))
		}

		fmt.Fprintln(os.Stderr)
	}

	if cfg.Verbose && !cfg.Quiet {
		modules := countModules(actx.GoFiles)
		fmt.Fprintf(os.Stderr, "Modules: %d  Detectors: %d  Findings: %d (before filtering)\n\n",
			modules, len(detectors), len(allFindings))
		fmt.Fprintf(os.Stderr, "Feature profile:\n%s\n", actx.FeatureProfile.String())
		if len(actx.LoadErrors) > 0 {
			fmt.Fprintf(os.Stderr, "Load errors (%d):\n", len(actx.LoadErrors))
			printLoadErrors(os.Stderr, actx.LoadErrors)
			fmt.Fprintln(os.Stderr)
		}
		printDetectorTimings(os.Stderr, result.Metrics)
	}

	if err := outputFindings(ctx, activeFindings, cfg); err != nil {
		return fmt.Errorf("output: %w", err)
	}

	if cfg.ShowSuppressed && len(suppressedFindings) > 0 && !cfg.Quiet {
		formatSuppressedFindings(os.Stdout, suppressedFindings, parseColorMode(cfg.Color))
	}

	if cfg.HealthScore {
		infoCap := cfg.Health.InfoCap
		if infoCap == 0 {
			infoCap = defaultInfoDeductionCap
		}
		hs := ComputeHealthScoreWithCap(unsuppressedFindings, infoCap)
		fmt.Print(renderHealthScore(hs, parseColorMode(cfg.Color)))
	}

	return shouldExitWithError(cfg, activeFindings)
}
