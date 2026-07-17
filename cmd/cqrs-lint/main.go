// Command cqrs-lint is a domain-aware linter for go-cqrs-lite consumers.
package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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

const version = "0.2.0"

// errFindingsWithErrors signals that error-severity findings were found.
// Returned from run() so cmdguard sets a non-zero exit code.
var errFindingsWithErrors = errors.New("findings with error severity")

// AppConfig holds all CLI configuration via cmdguard struct tags.
//
//nolint:tagalign,golines // tagalign and golines conflict on struct tag alignment
type AppConfig struct {
	cmdguard.Config

	Path           string `default:"."     flag:"path"            help:"Path to lint"`
	Format         string `default:"text"  flag:"format"          help:"Output format"                                              short:"o"`
	MinSeverity    string `default:"info"  flag:"min-severity"    help:"Minimum severity"`
	MinConfidence  string `default:"low"   flag:"min-confidence"  help:"Minimum confidence"`
	Fix            bool   `default:"false" flag:"fix"             help:"Apply auto-fixes"`
	DryRun         bool   `default:"false" flag:"dry-run"         help:"Show fixes without applying"`
	FastMode       bool   `default:"false" flag:"fast"            help:"Critical correctness rules only"`
	HealthScore    bool   `default:"false" flag:"health-score"    help:"Print only the health score"`
	Categories     string `default:""      flag:"only"            help:"Filter by category or rule IDs"`
	Exclude        string `default:""      flag:"exclude"         help:"Exclude paths (comma-separated)"`
	Color          string `default:"auto"  flag:"color"           help:"Colored output: auto,always,never"`
	Verbose        bool   `default:"false" flag:"verbose"         help:"Verbose output"`
	Quiet          bool   `default:"false" flag:"quiet"           help:"Suppress non-finding output"                                short:"q"`
	FPSuspects     bool   `default:"false" flag:"fp-suspects"     help:"Show only low-confidence findings (likely false positives)"`
	ShowSuppressed bool   `default:"false" flag:"show-suppressed" help:"Show suppressed findings with their suppression reason"`

	// Features declares which go-cqrs-lite modules the consumer uses.
	// Each non-nil flag overrides auto-detection. See FeatureProfile docs.
	Features analyzer.ConfigFeatures `json:"features,omitempty"`
	// Preset is a named set of feature-flag defaults (sugar over Features).
	// Explicit Features flags always override preset values.
	Preset analyzer.ConfigPreset `default:"" json:"preset,omitempty"`
	// Rules carries rule-specific overrides (e.g. external-API struct prefixes
	// for D002). See analyzer.RulesConfig docs for each field.
	Rules analyzer.RulesConfig `json:"rules,omitempty"`
	// Health carries health-score tuning (e.g. the Info-deduction cap).
	Health HealthConfig `json:"health,omitempty"`
}

// HealthConfig tunes the health-score computation. All fields default to zero,
// which preserves the standard scoring behavior.
//
//	{"health": {"info-cap": 15}}
//
// InfoCap caps the total penalty from Info-severity findings. 0 means use the
// built-in default (20). A negative value is treated as 0 (no cap).
type HealthConfig struct {
	InfoCap int `json:"info-cap,omitempty"`
}

func main() {
	cli, err := cmdguard.NewCLI[AppConfig](
		"cqrs-lint",
		"Domain-aware linter for go-cqrs-lite consumers",
		AppConfig{},
		cmdguard.WithCLIVersion(version),
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

	if err := setupDoctorCommand(cli); err != nil {
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

// shouldExitWithError determines the exit-code decision based on active
// findings and mode. Returns nil for success, errFindingsWithErrors for
// failure. In --fp-suspects mode, always returns nil (advisory mode).
func shouldExitWithError(cfg *AppConfig, activeFindings []finding.Finding) error {
	// --fp-suspects is advisory: never exit non-zero based on suspect findings.
	if cfg.FPSuspects {
		return nil
	}

	for _, f := range activeFindings {
		if f.Severity.Compare(finding.SeverityError) >= 0 {
			return errFindingsWithErrors
		}
	}

	return nil
}

func countModules(files []*analyzer.GoFile) int {
	seen := make(map[string]bool)
	for _, f := range files {
		seen[filepath.Dir(f.Path)] = true
	}
	return len(seen)
}

// loadRawRulesJSON reads .cqrs-lint.json from the current directory and returns
// the raw JSON of the "rules" key (for unknown-key validation). Returns nil if
// the file is absent, unreadable, or has no "rules" key — validation is
// best-effort and must never block a lint run.
func loadRawRulesJSON() []byte {
	data, err := os.ReadFile(".cqrs-lint.json")
	if err != nil {
		return nil
	}

	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		return nil
	}

	if rules, ok := top["rules"]; ok {
		if reencoded, err := json.Marshal(rules); err == nil {
			return reencoded
		}
	}

	return nil
}

func printDetectorTimings(w io.Writer, snap pipeline.MetricsSnapshot) {
	if len(snap.DetectorTimes) == 0 {
		return
	}

	type detStat struct {
		name     string
		duration time.Duration
		findings int
	}

	stats := make([]detStat, 0, len(snap.DetectorTimes))
	for name, d := range snap.DetectorTimes {
		stats = append(stats, detStat{name: name, duration: d, findings: snap.FindingsFound[name]})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].duration > stats[j].duration
	})

	_, _ = fmt.Fprintln(w, "Detector timings (slowest first):")
	for _, s := range stats {
		if s.duration < time.Millisecond {
			continue
		}
		_, _ = fmt.Fprintf(
			w,
			"  %-40s %8s  %d findings\n",
			s.name,
			s.duration.Round(time.Millisecond),
			s.findings,
		)
	}
	_, _ = fmt.Fprintln(w)
}
