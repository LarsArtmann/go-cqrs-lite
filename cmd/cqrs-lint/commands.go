package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
)

// registerCommand wraps the create-and-add pattern shared by every subcommand:
// it surfaces NewCommand failures as "create <name> command" and AddCommand
// failures as "add <name> command".
func registerCommand[F any](
	cli *cmdguard.CLI[AppConfig],
	name string,
	cmd cmdguard.Command[AppConfig, F],
	err error,
) error {
	if err != nil {
		return fmt.Errorf("create %s command: %w", name, err)
	}
	if err := cmdguard.AddCommand(cli, cmd); err != nil {
		return fmt.Errorf("add %s command: %w", name, err)
	}
	return nil
}

func setupRulesCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, cmdguard.NoFlags](
		"rules",
		cmdguard.NoFlags{},
		func(_ context.Context, cfg *AppConfig, _ cmdguard.NoFlags) error {
			out, err := renderRulesTable(parseColorMode(cfg.Color))
			if err != nil {
				return fmt.Errorf("render rules: %w", err)
			}

			fmt.Println(out)

			return nil
		},
		cmdguard.WithShort("List all available rules"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "rules", cmd, err)
}

func setupVersionCommand(cli *cmdguard.CLI[AppConfig]) error {
	cmd, err := cmdguard.NewCommand[AppConfig, versionFlags](
		"version",
		versionFlags{},
		func(_ context.Context, _ *AppConfig, flags versionFlags) error {
			if flags.Verbose {
				fmt.Println(versionVerbose())
			} else {
				fmt.Println(versionString())
			}
			return nil
		},
		cmdguard.WithShort("Print version"),
		cmdguard.WithNoArgs(),
	)
	return registerCommand(cli, "version", cmd, err)
}

// versionFlags adds --verbose to the version subcommand.
type versionFlags struct {
	Verbose bool `default:"false" flag:"verbose" help:"Show Go version, OS/arch, and module path"`
}

// versionVerbose returns the full version string with build environment details.
func versionVerbose() string {
	var b strings.Builder
	b.WriteString(versionString())
	b.WriteString("\n  go:      " + runtime.Version())
	b.WriteString("\n  arch:    " + runtime.GOOS + "/" + runtime.GOARCH)
	b.WriteString("\n  module:  github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint")
	return b.String()
}

// versionString returns the full version string, including commit hash and
// build date when they were injected via ldflags. Local `go build` runs
// produce a bare "cqrs-lint X.Y.Z"; Nix builds include provenance.
func versionString() string {
	var parts []string

	if commitHash != "" {
		parts = append(parts, "commit: "+commitHash)
	}

	if buildDate != "" {
		parts = append(parts, "built: "+buildDate)
	}

	if len(parts) == 0 {
		return "cqrs-lint " + version
	}

	return fmt.Sprintf("cqrs-lint %s (%s)", version, strings.Join(parts, ", "))
}
