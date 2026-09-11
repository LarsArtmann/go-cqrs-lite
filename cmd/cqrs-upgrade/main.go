package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"
)

var (
	// errInvalidToFlag fires when --to is not valid semver.
	errInvalidToFlag = errors.New("--to is not a valid semver version")
	// errStrictViolations fires when --strict finds v5-removed API usage.
	errStrictViolations = errors.New("--strict: v5-removed API usage detected")
	// errArgsAfterDir fires when arguments follow the positional module
	// directory. Stdlib flag parsing stops at the first positional, so a
	// flag written after the dir is silently ignored (2026-09-11:
	// `cqrs-upgrade . --strict` ran as a plain report instead of the gate).
	errArgsAfterDir = errors.New(
		"unexpected arguments after the module directory (flags must precede it)")
	// errStrictScanFailed fires when --strict cannot prove cleanliness: the
	// deprecation scan itself failed, so v5-readiness is unproven. A gate
	// that silently passes when its scanner breaks is not a gate.
	errStrictScanFailed = errors.New("--strict: deprecation scan failed")
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cqrs-upgrade:", err)
		os.Exit(1)
	}
}

// config holds the parsed CLI options.
type config struct {
	dir       string
	dryRun    bool
	noBuild   bool
	strict    bool
	jsonOut   bool
	to        string
	workspace bool
}

// parseFlags builds the config from args. dir defaults to the working
// directory and must contain a go.mod (or, with --workspace, be the root of a
// multi-module tree).
func parseFlags(args []string) (config, error) {
	cfg := config{}

	fs := flag.NewFlagSet("cqrs-upgrade", flag.ContinueOnError)
	fs.BoolVar(&cfg.dryRun, "dry-run", false,
		"show planned bumps and the deprecation report without changing go.mod")
	fs.BoolVar(
		&cfg.noBuild,
		"no-build",
		false,
		"skip the go mod tidy + build + vet verification after bumping",
	)
	fs.BoolVar(&cfg.strict, "strict", false,
		"exit non-zero when v5-removed API usage is detected (v5-readiness gate)")
	fs.BoolVar(&cfg.jsonOut, "json", false,
		"machine-readable JSON output (bump plan + deprecations per module)")
	fs.StringVar(&cfg.to, "to", "",
		"clamp every bump target to at most this version (semver, e.g. v4.13.0); never downgrades")
	fs.BoolVar(&cfg.workspace, "workspace", false,
		"upgrade every module under dir (skips vendor/, testdata/, .git/)")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	if cfg.to != "" && !semver.IsValid(cfg.to) {
		return cfg, fmt.Errorf("%w: %q", errInvalidToFlag, cfg.to)
	}

	if fs.NArg() > 1 {
		return cfg, fmt.Errorf("%w: %s", errArgsAfterDir, strings.Join(fs.Args()[1:], " "))
	}

	cfg.dir = "."
	if fs.NArg() > 0 {
		cfg.dir = fs.Arg(0)
	}

	abs, err := filepath.Abs(cfg.dir)
	if err != nil {
		return cfg, fmt.Errorf("resolve %s: %w", cfg.dir, err)
	}

	cfg.dir = abs

	return cfg, nil
}

// run executes the upgrade pipeline: collect pins → resolve latest →
// (dry-run | edit + verify) → deprecation report. With --workspace the
// pipeline runs for every go.mod under the root; with --strict a non-empty
// deprecation report fails the run; with --json the whole result is emitted
// as one machine-readable document.
func run(_ context.Context, args []string) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}

	dirs := []string{cfg.dir}

	if cfg.workspace {
		dirs, err = findGoMods(cfg.dir)
		if err != nil {
			return err
		}
	}

	reports := make([]moduleReport, 0, len(dirs))

	for _, dir := range dirs {
		reports = append(reports, upgradeModule(cfg, dir))
	}

	if cfg.jsonOut {
		return emitJSON(os.Stdout, reports)
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

	return strictGateError(reports)
}

// strictGateError returns the --strict failure for a report set: a failed
// deprecation scan fails first (an unscannable module is not proven clean),
// then actual v5-removed API usage.
func strictGateError(reports []moduleReport) error {
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

		return rep
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

	findings, scanErr := deprecationFindings(dir)
	rep.Deprecations = findings
	if scanErr != nil {
		rep.ScanErr = scanErr.Error()
	}

	return rep
}

// printReport renders one module's human-readable output.
func printReport(r moduleReport) {
	switch {
	case r.Error != "":
		fmt.Printf("error: %s\n", r.Error)
	case r.NoPins:
		fmt.Printf("no direct go-cqrs-lite pins found in %s\n", filepath.Join(r.Dir, "go.mod"))
	default:
		printBumps(r.Bumps)
		if r.ScanErr != nil {
			fmt.Printf("deprecation report: scan failed — %v (v5-readiness unknown)\n", r.ScanErr)
		} else {
			printDeprecations(os.Stdout, r.Deprecations)
		}
	}
}

// printBumps renders the plan table for both dry-run and apply runs.
func printBumps(bumps []bump) {
	fmt.Println(strings.ReplaceAll(formatBumps(bumps), "\t", "  "))
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

// emitJSON writes the full report as one deterministic JSON document.
// Struct-field order (no maps) keeps the output byte-stable for CI diffs.
func emitJSON(w io.Writer, reports []moduleReport) error {
	wire := make([]moduleJSON, 0, len(reports))

	for _, r := range reports {
		wire = append(wire, r.toJSON())
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(wire)
}
