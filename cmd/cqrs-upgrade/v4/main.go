package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func main() {
	cli, err := buildCLI()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cqrs-upgrade:", err)
		os.Exit(1)
	}

	cli.ExecuteAndExit(context.Background())
}

// AppConfig is the CLI flag surface: one typed field per flag, with help
// strings byte-identical to the pre-cmdguard stdlib flag set (--help parity,
// plan T19/m60). It deliberately does NOT embed cmdguard.Config: the
// batteries' config struct carries its own --strict flag, which would
// collide with this tool's load-bearing v5-readiness gate of the same name.
type AppConfig struct {
	DryRun    bool   `default:"false" flag:"dry-run"   help:"show planned bumps and the deprecation report without changing go.mod"`                                                                                                                                    //nolint:lll // struct tag must stay on one line
	NoBuild   bool   `default:"false" flag:"no-build"  help:"skip the go mod tidy + build + vet verification after bumping"`                                                                                                                                            //nolint:lll // struct tag must stay on one line
	Strict    bool   `default:"false" flag:"strict"    help:"exit non-zero on v5-removed API usage, failed scans, or module errors (v5-readiness gate)"`                                                                                                                //nolint:lll // struct tag must stay on one line
	JSONOut   bool   `default:"false" flag:"json"      help:"machine-readable JSON output (bump plan + deprecations per module)"`                                                                                                                                       //nolint:lll // struct tag must stay on one line
	To        string `default:""      flag:"to"        help:"clamp every bump target to at most this version (semver, e.g. v4.13.0); never downgrades"`                                                                                                                 //nolint:lll // struct tag must stay on one line
	Workspace bool   `default:"false" flag:"workspace" help:"upgrade every module under dir (skips vendor/, testdata/, .git/)"`                                                                                                                                         //nolint:lll // struct tag must stay on one line
	ToV5      bool   `default:"false" flag:"to-v5"     help:"v5 migration pass: rewrite imports of the folded v4 trains to their core/v5 twins (ADR-0152); go.mod requires are reported, not edited (core/v5 is untagged); combine with --dry-run for a per-file diff"` //nolint:lll // struct tag must stay on one line
}

// buildCLI wires the cqrs-upgrade CLI as a flat single-command tool: the
// seven pipeline flags live on the root, the module directory is the one
// optional positional. Extracted from main so CLI-level tests drive the
// exact production wiring via ExecuteWithArgs.
func buildCLI() (*cmdguard.CLI[AppConfig], error) {
	cli, err := cmdguard.NewCLI(
		"cqrs-upgrade",
		"Bump go-cqrs-lite module pins to the latest tags and report v5-removed API usage",
		AppConfig{},
	)
	//art-dupl:accept cobra CLI bootstrap guard — identical by design across cmd tools
	if err != nil {
		return nil, err
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "cqrs-upgrade [flags] [dir]"
	// The dir is a positional argument, not a subcommand. Without
	// ArbitraryArgs, cobra's default Args validator (active because the
	// battery commands register as subcommands) rejects every positional as
	// "Unknown command" — the documented `cqrs-upgrade <dir>` invocation
	// would be dead code.
	rootCmd.Args = cobra.ArbitraryArgs
	// Preserve the stdlib-flag parse contract the 2026-09-11 hardening
	// pinned: flags must precede the directory. pflag defaults to
	// interspersed parsing (a flag written after the dir would silently
	// apply); disabling it keeps `cqrs-upgrade . --strict` a LOUD
	// errArgsAfterDir failure instead of a second, subtler dialect.
	rootCmd.Flags().SetInterspersed(false)

	rootCmd.RunE = func(c *cobra.Command, args []string) error {
		cfg, err := configFromFlags(*cli.Config(), args)
		if err != nil {
			return err
		}

		return runPipeline(c.Context(), cfg)
	}

	return cli, nil
}

// config holds the validated pipeline options.
type config struct {
	dir       string
	dryRun    bool
	noBuild   bool
	strict    bool
	jsonOut   bool
	to        string
	workspace bool
	toV5      bool
}

// configFromFlags validates the parsed flag values against the positional
// arguments and builds the pipeline config: --to must be valid semver, at
// most one directory may follow the flags, and the directory resolves to an
// absolute path (default: the working directory, which must contain a
// go.mod — or, with --workspace, be the root of a multi-module tree).
func configFromFlags(ac AppConfig, args []string) (config, error) {
	cfg := config{
		dryRun:    ac.DryRun,
		noBuild:   ac.NoBuild,
		strict:    ac.Strict,
		jsonOut:   ac.JSONOut,
		to:        ac.To,
		workspace: ac.Workspace,
		toV5:      ac.ToV5,
	}

	if cfg.to != "" && !semver.IsValid(cfg.to) {
		return cfg, fmt.Errorf("%w: %q", errInvalidToFlag, cfg.to)
	}

	if len(args) > 1 {
		return cfg, fmt.Errorf("%w: %s", errArgsAfterDir, strings.Join(args[1:], " "))
	}

	cfg.dir = "."
	if len(args) > 0 {
		cfg.dir = args[0]
	}

	abs, err := filepath.Abs(cfg.dir)
	if err != nil {
		return cfg, fmt.Errorf("resolve %s: %w", cfg.dir, err)
	}

	cfg.dir = abs

	return cfg, nil
}

// runPipeline executes the upgrade pipeline: collect pins → resolve latest →
// (dry-run | edit + verify) → deprecation report. With --workspace the
// pipeline runs for every go.mod under the root; with --strict a non-empty
// deprecation report fails the run; with --json the whole result is emitted
// as one machine-readable document.
func runPipeline(_ context.Context, cfg config) error {
	dirs := []string{cfg.dir}

	if cfg.workspace {
		found, err := findGoMods(cfg.dir)
		if err != nil {
			return err
		}

		dirs = found
	}

	if cfg.toV5 {
		return runV5Mode(cfg, dirs)
	}

	reports := make([]moduleReport, 0, len(dirs))

	for _, dir := range dirs {
		reports = append(reports, upgradeModule(cfg, dir))
	}

	// Compute the strict verdict BEFORE branching on output format:
	// --json used to return early and silently skip strictGateError, making
	// `--json --strict` a no-op gate (E2E-caught, 2026-09-25).
	strictErr := strictGateError(reports)

	if cfg.jsonOut {
		if err := emitJSON(os.Stdout, reports); err != nil {
			return err
		}

		return strictErr
	}

	for i, r := range reports {
		if cfg.workspace {
			fmt.Printf("== %s ==\n", r.Dir)
		}

		printReport(r)

		if i < len(reports)-1 {
			fmt.Println()
		}
	}

	if cfg.dryRun {
		fmt.Println("dry run: no changes written")
	}

	return strictErr
}

// strictGateError returns the --strict failure for a report set: an
// errored module fails first (its pipeline died before any scan —
// unscanned is unproven), then a failed deprecation scan, then actual
// v5-removed API usage.
func strictGateError(reports []moduleReport) error {
	if n := countModuleErrors(reports); n > 0 {
		return fmt.Errorf("%w in %d module(s) — unscanned = unproven, see report",
			errStrictModuleFailed, n)
	}

	if n := countScanFailures(reports); n > 0 {
		return fmt.Errorf("%w in %d module(s) — v5-readiness unproven, see report",
			errStrictScanFailed, n)
	}

	if hasStrictViolation(reports) {
		return fmt.Errorf("%w in %d module(s) — see deprecation report",
			errStrictViolations, countStrictViolations(reports))
	}

	return nil
}

// countModuleErrors counts modules whose pipeline errored (rep.Error set:
// pin collection, go.mod edit, or build verification failure).
func countModuleErrors(reports []moduleReport) int {
	n := 0

	for _, r := range reports {
		if r.Error != "" {
			n++
		}
	}

	return n
}

// countScanFailures counts modules whose deprecation scan failed.
func countScanFailures(reports []moduleReport) int {
	n := 0

	for _, r := range reports {
		if r.ScanErr != nil {
			n++
		}
	}

	return n
}

// upgradeModule runs the full pipeline for one module directory and records
// everything the human and JSON outputs need.
func upgradeModule(cfg config, dir string) moduleReport {
	rep := moduleReport{Dir: dir}

	modPath := filepath.Join(dir, "go.mod")

	pins, err := collectPins(modPath)
	if err != nil {
		rep.Error = err.Error()

		return rep
	}

	if len(pins) == 0 {
		rep.NoPins = true

		// Still scan: an indirect-only cqrs consumer (no direct pins) can
		// use v5-removed APIs just the same — no pins ≠ no cqrs code
		// (strict-gate hole (a), 05-26 §e4). Bump planning is skipped, the
		// deprecation report is not.
		return scanDeprecations(dir, rep)
	}

	rep.Bumps = planUpgrades(pins, cfg.to)

	if !cfg.dryRun {
		if err := editGoMod(modPath, rep.Bumps); err != nil {
			rep.Error = err.Error()

			return rep
		}

		if !cfg.noBuild {
			if err := verify(dir); err != nil {
				rep.Error = fmt.Sprintf(
					"%v\nverification failed — go.mod was already bumped; "+
						"fix the code or pin back manually", err,
				)

				return rep
			}
		}
	}

	return scanDeprecations(dir, rep)
}

// scanDeprecations fills the report's deprecation-scan and suggestion
// fields and returns it. Shared by the no-pins early exit and the post-bump
// path — both owe the caller a v5-readiness report. Suggestions ride the
// same BuildContext (one package load): they never gate, they coach.
func scanDeprecations(dir string, rep moduleReport) moduleReport {
	findings, suggestions, scanErr := scanFindings(dir)
	rep.Deprecations = findings
	rep.Suggestions = suggestions
	rep.ScanErr = scanErr

	return rep
}

// hasStrictViolation reports whether any module has v5-removed API usage.
func hasStrictViolation(reports []moduleReport) bool {
	return countStrictViolations(reports) > 0
}

// countStrictViolations counts modules with at least one deprecation finding.
func countStrictViolations(reports []moduleReport) int {
	n := 0

	for _, r := range reports {
		if len(r.Deprecations) > 0 {
			n++
		}
	}

	return n
}
