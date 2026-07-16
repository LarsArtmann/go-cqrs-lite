// Command cqrs-lint is a domain-aware linter for go-cqrs-lite consumers.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/fix"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/suppression"
)

const version = "0.1.0"

// Config holds all CLI configuration.
type Config struct {
	Path        string
	Format      string
	MinSeverity string
	Fix         bool
	DryRun      bool
	FastMode    bool
	HealthScore bool
	Categories  []string
	Verbose     bool
	Quiet       bool
}

func main() {
	cfg := parseFlags()
	if cfg == nil {
		return
	}

	ctx := context.Background()
	if err := run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() *Config {
	args := os.Args[1:]
	cfg := &Config{
		Path:        ".",
		Format:      "text",
		MinSeverity: "info",
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			printHelp()

			return nil
		case "-v", "--version":
			fmt.Printf("cqrs-lint %s\n", version)

			return nil
		case "--fix":
			cfg.Fix = true
		case "--dry-run":
			cfg.DryRun = true
		case "--fast":
			cfg.FastMode = true
		case "--health-score":
			cfg.HealthScore = true
		case "--verbose":
			cfg.Verbose = true
		case "--quiet", "-q":
			cfg.Quiet = true
		case "-o", "--format":
			if i+1 < len(args) {
				cfg.Format = args[i+1]
				i++
			}
		case "--min-severity":
			if i+1 < len(args) {
				cfg.MinSeverity = args[i+1]
				i++
			}
		case "--only":
			if i+1 < len(args) {
				cfg.Categories = strings.Split(args[i+1], ",")
				i++
			}
		case "rules":
			fmt.Print(rules.ListRules())

			return nil
		case "version":
			fmt.Printf("cqrs-lint %s\n", version)

			return nil
		default:
			if !strings.HasPrefix(arg, "-") {
				cfg.Path = arg
			}
		}
	}

	absPath, err := filepath.Abs(cfg.Path)
	if err == nil {
		cfg.Path = absPath
	}

	return cfg
}

func printHelp() {
	fmt.Print(`cqrs-lint — Domain-aware linter for go-cqrs-lite consumers

Usage:
  cqrs-lint [path] [flags]

Commands:
  cqrs-lint lint [path]     Lint Go project for CQRS anti-patterns (default)
  cqrs-lint rules           List all available rules
  cqrs-lint version         Print version

Flags:
  --format <fmt>            Output format: text, json, sarif, markdown (default: text)
  --min-severity <sev>      Minimum severity: info, warning, error, critical (default: info)
  --fix                     Apply auto-fixes
  --dry-run                 Show fixes without applying
  --fast                    Run only Critical/High correctness rules
  --health-score            Print only the health score
  --only <cats>             Run only specific categories (comma-separated)
  --verbose                 Verbose output
  --quiet                   Suppress non-finding output
  -h, --help                Show help
  -v, --version             Print version

Examples:
  cqrs-lint ./...
  cqrs-lint ./internal/... --format json
  cqrs-lint --fix --dry-run ./...
  cqrs-lint --fast ./...
  cqrs-lint --health-score ./...
`)
}

func run(ctx context.Context, cfg *Config) error {
	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	start := time.Now()

	// Build analysis context.
	actx, err := analyzer.BuildContext(absPath)
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	if len(actx.GoFiles) == 0 {
		if !cfg.Quiet {
			fmt.Fprintln(os.Stderr, "No Go files importing go-cqrs-lite found. Nothing to lint.")
		}

		return nil
	}

	// Select detectors.
	var detectors []finding.Detector
	if cfg.FastMode {
		detectors = rules.RegisterCritical(actx)
	} else {
		detectors = rules.RegisterAll(actx)
		if len(cfg.Categories) > 0 {
			detectors = rules.FilterByCategory(detectors, cfg.Categories)
		}
	}

	// Build pipeline config.
	// DryRun is true unless --fix is explicitly passed — we never auto-apply fixes.
	pipeConfig := pipeline.Config{
		MaxIterations:       1,
		ParallelDetectors:   true,
		GracefulDegradation: true,
		DryRun:              !cfg.Fix, // detection-only unless --fix
		Timeout:             5 * time.Minute,
		Processors: []pipeline.FindingTransformer{
			suppression.NewSuppressionFilter(),
		},
	}

	if cfg.Fix || cfg.DryRun {
		pipeConfig.FixProviders = []pipeline.FixProvider{fix.NewCQRSFixProvider()}
	}

	// Run pipeline.
	pipe, err := pipeline.New(pipeConfig, absPath, detectors...)
	if err != nil {
		return fmt.Errorf("create pipeline: %w", err)
	}

	result, err := pipe.Run(ctx)
	if err != nil {
		return fmt.Errorf("pipeline run: %w", err)
	}

	// Collect findings.
	allFindings := collectFindings(result)
	activeFindings := filterBySeverity(allFindings, cfg.MinSeverity)

	// Output.
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

	if err := outputFindings(activeFindings, cfg); err != nil {
		return fmt.Errorf("output: %w", err)
	}

	// Exit non-zero if there are error/critical findings.
	for _, f := range activeFindings {
		if f.Severity >= finding.SeverityError {
			os.Exit(1)
		}
	}

	return nil
}

func collectFindings(result *pipeline.PipelineResult) []finding.Finding {
	var all []finding.Finding
	for _, iter := range result.Iterations {
		all = append(all, iter.Findings()...)
	}

	if result.Verification != nil {
		all = append(all, result.Verification.Remaining...)
		all = append(all, result.Verification.NewFindings...)
	}
	// Deduplicate by ID.
	seen := make(map[finding.ID]bool)

	var unique []finding.Finding

	for _, f := range all {
		if seen[f.ID] {
			continue
		}

		seen[f.ID] = true
		unique = append(unique, f)
	}

	return unique
}

func filterBySeverity(findings []finding.Finding, minSev string) []finding.Finding {
	minS := parseSeverity(minSev)

	var result []finding.Finding

	for _, f := range findings {
		if f.Severity >= minS {
			result = append(result, f)
		}
	}

	return result
}

func parseSeverity(s string) finding.Severity {
	switch strings.ToLower(s) {
	case "critical":
		return finding.SeverityCritical
	case "error":
		return finding.SeverityError
	case "warning":
		return finding.SeverityWarning
	case "info":
		return finding.SeverityInfo
	default:
		return finding.SeverityInfo
	}
}

func outputFindings(findings []finding.Finding, cfg *Config) error {
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
		err := report.WriteSARIF(context.Background(), os.Stdout)
		if err != nil {
			return err
		}

	case "markdown":
		err := finding.FormatMarkdown(os.Stdout, findings)
		if err != nil {
			return err
		}

	default: // text
		if len(findings) == 0 {
			if !cfg.Quiet {
				fmt.Println("No findings. Clean!")
			}

			return nil
		}

		return finding.FormatText(os.Stdout, findings)
	}

	return nil
}
