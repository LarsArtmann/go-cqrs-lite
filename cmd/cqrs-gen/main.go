// cqrs-gen generates typed handler registration code from Go structs marked with
// cqrs annotations.
//
// Usage:
//
//	cqrs-gen -type=command -output=commands_gen.go ./...
//	cqrs-gen -type=query -output=queries_gen.go ./...
//	cqrs-gen -type=event -output=events_gen.go ./...
//
// Marker comments in source code (the identifier after the kind becomes the
// registered command/query/event type):
//
//	//cqrs:command CreateUser
//	//cqrs:query GetUser
//	//cqrs:event UserCreated
//	type CreateUserCmd struct {
//	    *command.BasicCommand
//	    Name string
//	}
package main

import (
	"context"
	"fmt"
	"os"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/spf13/cobra"
)

const (
	genTypeCommand = "command"
	genTypeQuery   = "query"
	genTypeEvent   = "event"
)

type AppConfig struct {
	cmdguard.Config

	Type   string `default:"command"         flag:"type"   help:"handler type to generate: command, query, or event"`
	Output string `default:"handlers_gen.go" flag:"output" help:"output file path"`
	Pkg    string `default:""                flag:"pkg"    help:"package name for generated file (defaults to the scanned source package)"`
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
		"Generate typed handler registration code from cqrs annotations",
		AppConfig{},
		cmdguard.WithCLILong(
			"cqrs-gen generates typed handler registration code from Go structs marked with cqrs annotations.\n\n"+
				"Marker comments in source code:\n"+
				"  //cqrs:command CreateUser\n"+
				"  //cqrs:query GetUser\n"+
				"  //cqrs:event UserCreated",
		),
	)
	//art-dupl:accept cobra CLI bootstrap guard — identical by design across cmd tools
	if err != nil {
		return nil, err
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "cqrs-gen [-type=command] [-output=handlers_gen.go] [paths...]"
	// ArbitraryArgs: the positional arguments are SCAN PATHS, not
	// subcommands. Without this, cobra's default validator (active because
	// help/completion are registered subcommands) rejects every positional
	// as "Unknown command" — the documented `cqrs-gen ./...` invocation was
	// unreachable dead code.
	rootCmd.Args = cobra.ArbitraryArgs
	rootCmd.RunE = func(_ *cobra.Command, args []string) error {
		cfg := cli.Config()

		paths := args
		if len(paths) == 0 {
			paths = []string{"."}
		}

		return run(cfg.Type, cfg.Output, cfg.Pkg, paths)
	}

	return cli, nil
}

func run(handlerType, outputFile, pkg string, paths []string) error {
	if handlerType != genTypeCommand && handlerType != genTypeQuery && handlerType != genTypeEvent {
		return fmt.Errorf("invalid type %q: must be 'command', 'query', or 'event'", handlerType)
	}

	entries, err := scan(paths, handlerType)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no cqrs markers found")
		return nil
	}

	entries = dedupEntries(os.Stderr, entries)

	if pkg == "" {
		pkg = entries[0].PackageName
	}

	code, err := generate(pkg, handlerType, entries)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if err := os.WriteFile(outputFile, []byte(code), 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	fmt.Printf("generated %d handlers → %s\n", len(entries), outputFile)
	return nil
const (
	commandImports = `import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

`

	queryImports = `import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

`

	eventImports = `import (
	"context"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

`
)
