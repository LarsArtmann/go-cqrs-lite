// cqrs-gen derives the wiring around marked command, query and event structs:
// typed registration for the composition root, typed event constructors, and
// optional catalog declarations. One cqrs_gen.go is written per package.
//
// Usage:
//
//	cqrs-gen --type=all --target=system .
//	cqrs-gen --catalog ./...
//	cqrs-gen --check .
//
// Marker comments in source code (the value is the registered wire name):
//
//	//cqrs:command task.create
//	//cqrs:query task.get
//	//cqrs:event task.created
//	type CreateTaskCmd struct {
//	    *command.BasicCommand
//	    Title string
//	}
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"
	"github.com/spf13/cobra"
)

type AppConfig struct {
	cmdguard.Config

	Type     string `default:"all"    flag:"type"     help:"kinds: command, query, event, all, or a comma list"`
	Target   string `default:"system" flag:"target"   help:"registration shape: system or dispatcher"`
	Output   string `default:""       flag:"output"   help:"one output file for a single package"`
	Pkg      string `default:""       flag:"pkg"      help:"package name of the --output file"`
	Catalog  bool   `default:"false"  flag:"catalog"  help:"declare every marked message for the catalog"`
	Check    bool   `default:"false"  flag:"check"    help:"fail when a generated file is missing or stale"`
	Handlers bool   `default:"false"  flag:"handlers" help:"also generate typed event subscription handlers"`
	List     bool   `default:"false"  flag:"list"     help:"print the discovered markers without writing"`
}

func main() {
	cli, err := buildCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI: %v\n", err)
		os.Exit(1)
	}

	cli.ExecuteAndExit(context.Background())
}

// buildCLI wires the cqrs-gen CLI (root command with positional scan paths).
// Extracted from main so the CLI-level regression tests can drive the exact
// production wiring via ExecuteWithArgs.
func buildCLI() (*cmdguard.CLI[AppConfig], error) {
	cli, err := cmdguard.NewCLI(
		"cqrs-gen",
		"Generate command, query and event wiring from cqrs markers",
		AppConfig{},
		cmdguard.WithCLILong(
			"cqrs-gen derives command, query and event wiring from Go structs marked with cqrs annotations.\n\n"+
				"Marker comments in source code:\n"+
				"  //cqrs:command task.create\n"+
				"  //cqrs:query task.get\n"+
				"  //cqrs:event task.created",
		),
	)
	//art-dupl:accept cobra CLI bootstrap guard — identical by design across cmd tools
	if err != nil {
		return nil, err
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "cqrs-gen [paths...]"
	// ArbitraryArgs: the positional arguments are SCAN PATHS, not
	// subcommands. Without this, cobra's default validator (active because
	// help/completion are registered subcommands) rejects every positional
	// as "Unknown command" — the documented `cqrs-gen ./...` invocation was
	// unreachable dead code.
	rootCmd.Args = cobra.ArbitraryArgs
	rootCmd.RunE = func(_ *cobra.Command, args []string) error {
		paths := args
		if len(paths) == 0 {
			paths = []string{"."}
		}

		return run(*cli.Config(), paths)
	}

	return cli, nil
}

func run(cfg AppConfig, paths []string) error {
	kinds, err := parseKinds(cfg.Type)
	if err != nil {
		return err
	}

	target, err := parseTarget(cfg.Target)
	if err != nil {
		return err
	}

	if cfg.Pkg != "" && cfg.Output == "" {
		return errors.New("--pkg only applies together with --output: " +
			"per-package output keeps each source package's name")
	}

	pkgs, err := scan(paths, kinds)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	opts := genOptions{target: target, catalog: cfg.Catalog, handlers: cfg.Handlers}

	switch {
	case cfg.List:
		return listMarkers(os.Stdout, pkgs)
	case cfg.Check:
		return checkGenerated(pkgs, opts, cfg.Output, cfg.Pkg)
	default:
		return writeGenerated(pkgs, opts, cfg.Output, cfg.Pkg)
	}
}
