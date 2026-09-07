package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cqrs-upgrade:", err)
		os.Exit(1)
	}
}

// config holds the parsed CLI options.
type config struct {
	dir     string
	dryRun  bool
	noBuild bool
}

// parseFlags builds the config from args. dir defaults to the working
// directory and must contain a go.mod.
func parseFlags(args []string) (config, error) {
	cfg := config{}

	fs := flag.NewFlagSet("cqrs-upgrade", flag.ContinueOnError)
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "show planned bumps without changing go.mod")
	fs.BoolVar(&cfg.noBuild, "no-build", false, "skip the go mod tidy + build + vet verification after bumping")

	if err := fs.Parse(args); err != nil {
		return cfg, err
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
// (dry-run | edit + verify) → deprecation report.
func run(_ context.Context, args []string) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}

	modPath := filepath.Join(cfg.dir, "go.mod")

	pins, err := collectPins(modPath)
	if err != nil {
		return err
	}

	if len(pins) == 0 {
		fmt.Printf("no direct go-cqrs-lite pins found in %s\n", modPath)

		return nil
	}

	bumps := planUpgrades(pins)
	printBumps(bumps)

	if cfg.dryRun {
		fmt.Println("dry run: no changes written")

		return nil
	}

	if err := editGoMod(modPath, bumps); err != nil {
		return err
	}

	if !cfg.noBuild {
		fmt.Println("verifying (tidy + build + vet, GOWORK=off) ...")
		if err := verify(cfg.dir); err != nil {
			return fmt.Errorf("%w\nverification failed — go.mod was already bumped; "+
				"fix the code or pin back manually", err)
		}
	}

	deprecationReport(os.Stdout, cfg.dir)

	return nil
}

// printBumps renders the plan table for both dry-run and apply runs.
func printBumps(bumps []bump) {
	fmt.Println(strings.ReplaceAll(formatBumps(bumps), "\t", "  "))
}
