// Command cqrs-lint is a domain-aware linter for go-cqrs-lite consumers.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cmdguard "github.com/larsartmann/cmdguard/v3/pkg/cmdguard/v3"
	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
	"github.com/spf13/cobra"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/fix"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/suppression"
)

const version = "0.1.0"

// errFindingsWithErrors signals that error-severity findings were found.
// Returned from run() so cmdguard sets a non-zero exit code.
var errFindingsWithErrors = errors.New("findings with error severity")

// AppConfig holds all CLI configuration via cmdguard struct tags.
//
//nolint:tagalign,golines // tagalign and golines conflict on struct tag alignment
type AppConfig struct {
	cmdguard.Config

	Path          string `default:"." flag:"path" help:"Path to lint"`
	Format        string `default:"text" flag:"format" help:"Output format" short:"o"`
	MinSeverity   string `default:"info" flag:"min-severity" help:"Minimum severity"`
	MinConfidence string `default:"low" flag:"min-confidence" help:"Minimum confidence"`
	Fix           bool   `default:"false" flag:"fix" help:"Apply auto-fixes"`
	DryRun        bool   `default:"false" flag:"dry-run" help:"Show fixes without applying"`
	FastMode      bool   `default:"false" flag:"fast" help:"Critical correctness rules only"`
	HealthScore   bool   `default:"false" flag:"health-score" help:"Print only the health score"`
	Categories    string `default:"" flag:"only" help:"Filter by category or rule IDs"`
	Exclude       string `default:"" flag:"exclude" help:"Exclude paths (comma-separated)"`
	Color         string `default:"auto" flag:"color" help:"Colored output: auto,always,never"`
	Verbose       bool   `default:"false" flag:"verbose" help:"Verbose output"`
	Quiet         bool   `default:"false" flag:"quiet" help:"Suppress non-finding output" short:"q"`
}

func main() {
	cli, err := cmdguard.NewCLI[AppConfig](
		"cqrs-lint",
		"Domain-aware linter for go-cqrs-lite consumers",
		AppConfig{},
		cmdguard.WithCLIVersion(version),
		cmdguard.WithFang(false),
		cmdguard.WithConfigFile(".cqrs-lint.json"),
		cmdguard.WithCLILong(
			"cqrs-lint detects anti-patterns in projects consuming the go-cqrs-lite library.",
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI: %v\n", err)
		os.Exit(1)
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "cqrs-lint [path] [flags]"
	rootCmd.Long = "cqrs-lint — Domain-aware linter for go-cqrs-lite consumers\n\n" +
		"Analyzes Go projects for CQRS anti-patterns, correctness bugs, and API misuse.\n\n" +
		"Usage:\n" +
		"  cqrs-lint [path] [flags]     Lint Go project for CQRS anti-patterns (default)\n" +
		"  cqrs-lint rules              List all available rules\n" +
		"  cqrs-lint version            Print version\n"
	rootCmd.Args = cobra.MaximumNArgs(1)
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		cfg := cli.Config()

		path := cfg.Path
		if len(args) > 0 {
			path = args[0]
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		cfg.Path = absPath

		return run(cmd.Context(), cfg)
	}

	if err := setupRulesCommand(cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := setupVersionCommand(cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := setupInitCommand(cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	cli.ExecuteAndExit(ctx)
}

func run(ctx context.Context, cfg *AppConfig) error {
	start := time.Now()

	actx, err := analyzer.BuildContext(cfg.Path)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	if len(actx.GoFiles) == 0 {
		if !cfg.Quiet {
			fmt.Fprintln(os.Stderr, "No Go files importing go-cqrs-lite found. Nothing to lint.")
		}

		return nil
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
		MaxIterations:       1,
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
	activeFindings := filterBySeverity(allFindings, cfg.MinSeverity)
	activeFindings = filterByConfidence(activeFindings, cfg.MinConfidence)

	if cfg.HealthScore {
		hs := ComputeHealthScore(activeFindings)
		fmt.Print(FormatHealthScore(hs))

		return nil
	}

	if !cfg.Quiet && cfg.Format == "text" {
		elapsed := time.Since(start)
		fmt.Fprintf(
			os.Stderr,
			"Analyzed %d files in %s\n\n",
			len(actx.GoFiles),
			elapsed.Round(time.Millisecond),
		)
	}

	if err := outputFindings(ctx, activeFindings, cfg); err != nil {
		return fmt.Errorf("output: %w", err)
	}

	hasErrors := false

	for _, f := range activeFindings {
		if f.Severity.Compare(finding.SeverityError) >= 0 {
			hasErrors = true

			break
		}
	}

	if hasErrors {
		return errFindingsWithErrors
	}

	return nil
}

func outputFindings(ctx context.Context, findings []finding.Finding, cfg *AppConfig) error {
	report := finding.NewReport(finding.ToolInfo{Name: "cqrs-lint", Version: version})
	report.AddFindings(findings)

	switch strings.ToLower(cfg.Format) {
	case "json":
		json, err := report.PrettyJSON()
		if err != nil {
			return err
		}

		fmt.Println(json)

	case "sarif":
		err := report.WriteSARIF(ctx, os.Stdout)
		if err != nil {
			return err
		}

	case "markdown":
		err := finding.FormatMarkdown(os.Stdout, findings)
		if err != nil {
			return err
		}

	default:
		if len(findings) == 0 {
			if !cfg.Quiet {
				fmt.Println("No findings. Clean!")
			}

			return nil
		}

		if cfg.Color == "always" || (cfg.Color != "never" && shouldUseColor(cfg.Color, os.Stdout)) {
			formatColoredText(os.Stdout, findings, cfg.Color)
			return nil
		}

		return finding.FormatText(os.Stdout, findings)
	}

	return nil
}
