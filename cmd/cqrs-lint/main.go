// Command cqrs-lint is a domain-aware linter for go-cqrs-lite consumers.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/larsartmann/go-finding"
	"github.com/spf13/cobra"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

const version = "4.3.0"

// commitHash and buildDate are injected via -ldflags at build time (Nix flake,
// CI). When empty (local `go build`), the version output omits them.
var (
	commitHash string //nolint:gochecknoglobals // injected via -ldflags
	buildDate  string //nolint:gochecknoglobals // injected via -ldflags
)

// errFindingsWithErrors signals that error-severity findings were found.
// Returned from run() so cmdguard sets a non-zero exit code.
var errFindingsWithErrors = errors.New("findings with error severity")

// AppConfig holds all CLI configuration via cmdguard struct tags.
type AppConfig struct {
	cmdguard.Config

	Path           string `default:"."     flag:"path"            help:"Path to lint"`
	Format         string `default:"text"  flag:"format"          help:"Output format"                                                      short:"o"`
	MinSeverity    string `default:"info"  flag:"min-severity"    help:"Minimum severity"`
	MinConfidence  string `default:"low"   flag:"min-confidence"  help:"Minimum confidence"`
	Fix            bool   `default:"false" flag:"fix"             help:"Apply auto-fixes"`
	DryRun         bool   `default:"false" flag:"dry-run"         help:"Show fixes without applying"`
	FastMode       bool   `default:"false" flag:"fast"            help:"Critical correctness rules only"`
	HealthScore    bool   `default:"false" flag:"health-score"    help:"Print only the health score"`
	Categories     string `default:""      flag:"only"            help:"Filter by category or rule IDs"`
	ExcludeRules   string `default:""      flag:"exclude-rules"   help:"Exclude rule IDs (comma-separated)"`
	Exclude        string `default:""      flag:"exclude"         help:"Exclude paths (comma-separated)"`
	Color          string `default:"auto"  flag:"color"           help:"Colored output: auto,always,never"`
	Verbose        bool   `default:"false" flag:"verbose"         help:"Verbose output"`
	GroupBy        string `default:""      flag:"group-by"        help:"Group findings by: none, module, aggregate"`
	Quiet          bool   `default:"false" flag:"quiet"           help:"Suppress non-finding output"                                        short:"q"`
	FPSuspects     bool   `default:"false" flag:"fp-suspects"     help:"Show only low-confidence findings (likely false positives)"`
	ShowSuppressed bool   `default:"false" flag:"show-suppressed" help:"Show suppressed findings with their suppression reason"`
	StrictLoad     bool   `default:"false" flag:"strict-load"     help:"Exit non-zero if any packages failed to load (partial analysis)"`
	Adoption       bool   `default:"false" flag:"adoption"        help:"Show F-series adoption coaching but exclude them from health score"`

	// Features declares which go-cqrs-lite modules the consumer uses.
	// Each non-nil flag overrides auto-detection. See FeatureProfile docs.
	Features analyzer.ConfigFeatures `json:"features,omitempty"` //nolint:modernize // config compatibility
	// Preset is a named set of feature-flag defaults (sugar over Features).
	// Explicit Features flags always override preset values.
	Preset analyzer.ConfigPreset `default:"" json:"preset,omitempty"`
	// Rules carries rule-specific overrides (e.g. external-API struct prefixes
	// for D002). See analyzer.RulesConfig docs for each field.
	Rules analyzer.RulesConfig `json:"rules,omitempty"` //nolint:modernize // config compatibility
	// Health carries health-score tuning (e.g. the Info-deduction cap).
	Health HealthConfig `json:"health,omitempty"` //nolint:modernize // config compatibility
}

// HealthConfig tunes the health-score computation. All fields default to zero,
// which preserves the standard scoring behavior.
//
//	{"health": {"info-cap": 15}}
//
// InfoCap caps the total penalty from Info-severity findings. 0 means use the
// built-in default (20). A negative value is treated as 0 (no cap).
type HealthConfig struct {
	InfoCap int `json:"info-cap,omitempty"` //nolint:tagliatelle // CLI config key
}

func main() {
	cli, err := cmdguard.NewCLI(
		"cqrs-lint",
		"Domain-aware linter for go-cqrs-lite consumers",
		AppConfig{},
		cmdguard.WithCLIVersion(version),
		cmdguard.WithConfigFileLoader(JSONCLoader{}, ".cqrs-lint.json"),
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
		"  cqrs-lint explain            Explain the .cqrs-lint.json config format in detail\n" +
		"  cqrs-lint doctor             Show resolved config + detected feature profile\n" +
		"  cqrs-lint init               Create a .cqrs-lint.json with defaults\n" +
		"  cqrs-lint version            Print version\n\n" +
		"SUPPRESSIONS:\n\n" +
		"  Inline (single rule):\n" +
		"    //cqrs-lint:ignore(C007) reason text\n\n" +
		"  Inline (multiple rules):\n" +
		"    //cqrs-lint:ignore(C007,A001) reason text\n\n" +
		"  Block:\n" +
		"    //cqrs-lint:ignore-start\n" +
		"    ...code...\n" +
		"    //cqrs-lint:ignore-end\n\n" +
		"  Block (specific rules):\n" +
		"    //cqrs-lint:ignore-start(C007,A001)\n" +
		"    ...code...\n" +
		"    //cqrs-lint:ignore-end\n\n" +
		"  Both //cqrs-lint: and // cqrs-lint: (with space) are accepted.\n" +
		"  Place inline suppressions on the line above the code or at end of line.\n" +
		"  Struct-field-level: place the comment directly above the field.\n\n" +
		"  Disable rules project-wide via config: {\"rules\": {\"disable\": [\"P012\"]}}\n" +
		"  or the --exclude-rules flag.\n" +
		"  Rule-specific config: {\"rules\": {\"external-api-struct-prefixes\": [\"Discord\"]}}.\n"
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

	if err := setupChangelogCommand(cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := setupExplainCommand(cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	cli.ExecuteAndExit(ctx)
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
