// Command cqrs-lint is a domain-aware linter for go-cqrs-lite consumers.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-finding"
	"github.com/spf13/cobra"
)

// resolvedVersion reports the build's version with NO hand-maintained
// constant, removing the const-drift class (v4.7.0 shipped while the const
// read 4.6.0; the v4.8.0 tagger's sed shipped an unquoted const — a syntax
// error). Precedence: `go install …@v4.x.y` records the true tag in
// Main.Version; in-repo builds carry vcs.revision ("dev-<sha>[-dirty]");
// Nix/ldflags builds see neither and fall back to "dev" — versionLine then
// appends the injected commitHash/buildDate.
func resolvedVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return strings.TrimPrefix(v, "v")
		}

		if rev, ok := buildInfoSetting(info, "vcs.revision"); ok {
			if len(rev) > 12 {
				rev = rev[:12]
			}

			if mod, _ := buildInfoSetting(info, "vcs.modified"); mod == "true" {
				rev += "-dirty"
			}

			return "dev-" + rev
		}
	}

	return "dev"
}

// moduleVersion reports the resolved dependency version for a module path
// from build info ("(devel)" and missing modules yield ""). `cqrs-lint
// version` uses it to surface which go-finding release the binary embeds —
// the corpus of behavior pins depends on that exact version.
func moduleVersion(path string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	for _, dep := range info.Deps {
		if dep.Path != path {
			continue
		}

		if dep.Version == "" || dep.Version == "(devel)" {
			return ""
		}

		return strings.TrimPrefix(dep.Version, "v")
	}

	return ""
}

// buildInfoSetting returns the named stamp from the build-info settings.
func buildInfoSetting(info *debug.BuildInfo, key string) (string, bool) {
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value, s.Value != ""
		}
	}

	return "", false
}

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
//
// Flag scoping contract (the subcommand-consistency rules):
//   - Fields WITHOUT local:"true" are persistent: they propagate to every
//     subcommand (path, format, color, min-severity, min-confidence,
//     typed-info). Only flags a subcommand actually reads may be persistent.
//   - Fields WITH local:"true" are lint-run-only. They do NOT appear on
//     subcommands — passing --fix to `cqrs-lint version` is an unknown-flag
//     error instead of a silently ignored no-op.
type AppConfig struct {
	cmdguard.Config

	Path          string `default:"."    flag:"path"           help:"Path to lint"`
	Format        string `default:"text" flag:"format"         help:"Output format: text, json, sarif, markdown, csv, tsv (per-command subsets)" short:"o"`
	MinSeverity   string `default:"info" flag:"min-severity"   help:"Minimum severity"`
	MinConfidence string `default:"low"  flag:"min-confidence" help:"Minimum confidence: none|low|medium|high|full or decimal; keeps >= floor"`
	Color         string `default:"auto" flag:"color"          help:"Colored output: auto,always,never"`

	Fix                     bool   `default:"false" flag:"fix"                        help:"Apply auto-fixes"                                                                                             local:"true"`
	DryRun                  bool   `default:"false" flag:"dry-run"                    help:"Show fixes without applying"                                                                                  local:"true"`
	FastMode                bool   `default:"false" flag:"fast"                       help:"Critical correctness rules only"                                                                              local:"true"`
	HealthScore             bool   `default:"false" flag:"health-score"               help:"Print only the health score"                                                                                  local:"true"`
	Categories              string `default:""      flag:"only"                       help:"Filter by category or rule IDs"                                                                               local:"true"`
	ExcludeRules            string `default:""      flag:"exclude-rules"              help:"Exclude rule IDs (comma-separated)"                                                                           local:"true"`
	Exclude                 string `default:""      flag:"exclude"                    help:"Exclude paths (comma-separated)"                                                                              local:"true"`
	Verbose                 bool   `default:"false" flag:"verbose"                    help:"Verbose output"                                                                                               local:"true"`
	GroupBy                 string `default:""      flag:"group-by"                   help:"Group findings by: none, module, aggregate"                                                                   local:"true" json:"group-by,omitempty"` //nolint:tagalign,tagliatelle
	Quiet                   bool   `default:"false" flag:"quiet"                      help:"Suppress non-finding output"                                                                                  local:"true"                           short:"q"`
	FPSuspects              bool   `default:"false" flag:"fp-suspects"                help:"Show only low-confidence findings (likely false positives)"                                                   local:"true"`
	ShowSuppressed          bool   `default:"false" flag:"show-suppressed"            help:"Show suppressed findings with their suppression reason"                                                       local:"true"`
	StrictLoad              bool   `default:"false" flag:"strict-load"                help:"Exit non-zero if any packages failed to load (partial analysis)"                                              local:"true"`
	FailOnStaleSuppressions bool   `default:"false" flag:"fail-on-stale-suppressions" help:"Exit non-zero if any //cqrs-lint:ignore directives are stale (not suppressing anything)"                      local:"true"`
	Adoption                bool   `default:"false" flag:"adoption"                   help:"Show F-series adoption coaching but exclude them from health score"                                           local:"true"`
	Scorecard               bool   `default:"false" flag:"scorecard"                  help:"Print module adoption scorecard (used/missing/coverage); the scorecard subcommand adds --scorecard-threshold" local:"true"`

	// Features declares which go-cqrs-lite modules the consumer uses.
	// Each non-nil flag overrides auto-detection. See FeatureProfile docs.
	Features analyzer.ConfigFeatures `json:"features,omitempty"` //nolint:modernize // config compatibility
	// Preset is a named set of feature-flag defaults (sugar over Features).
	// Explicit Features flags always override preset values.
	Preset analyzer.ConfigPreset `json:"preset,omitempty" default:""`
	// Rules carries rule-specific overrides (e.g. external-API struct prefixes
	// for D002). See analyzer.RulesConfig docs for each field.
	Rules analyzer.RulesConfig `json:"rules,omitempty"` //nolint:modernize // config compatibility
	// Health carries health-score tuning (e.g. the Info-deduction cap).
	Health HealthConfig `json:"health,omitempty"` //nolint:modernize // config compatibility
	// TypedInfo gates the F091 typed-confirmation tier: rules that need type
	// information to attribute or confirm findings (F090(b) dot-import
	// attribution, C008 usage confirmation) run their typed path only when
	// this allows it. "auto" (default) enables them whenever the package
	// load produced type info; "on" forces them on; "off" restores the
	// pre-typed name-only behavior everywhere.
	TypedInfo string `json:"typed-info,omitempty" default:"auto" flag:"typed-info" help:"Typed confirmation tier: auto, on, off"` //nolint:tagliatelle // CLI config key
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
		cmdguard.WithCLIVersion(resolvedVersion()),
		cmdguard.WithConfigFileLoader(JSONCLoader{}, ".cqrs-lint.json"),
		cmdguard.WithCLILong(
			"cqrs-lint detects anti-patterns in projects consuming the go-cqrs-lite library.",
		),
	)
	//art-dupl:accept cobra CLI bootstrap guard — identical by design across cmd tools
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI: %v\n", err)
		os.Exit(1)
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "cqrs-lint [path] [flags]"
	rootCmd.Long = rootLongHelp
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

	if err := setupScorecardCommand(cli); err != nil {
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
