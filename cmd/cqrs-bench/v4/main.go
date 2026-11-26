// Command cqrs-bench runs go-cqrs-lite benchmarks against any supported
// backend. It is the CLI front-end for the benchkit library.
//
// Usage:
//
//	cqrs-bench run --backend sqlite --profile dev
//	cqrs-bench compare --profile small --format markdown
//	cqrs-bench sweep --param workers --values 1,2,4,8 --backend memory
//
// Every backend is declared once, in the registry in backend.go; help text,
// validation, and alias resolution all derive from it. Handlers return
// errors instead of exiting, so a failed or interrupted command still
// removes its temporary stores and flushes its profiles.
package main

import (
	"context"
	"fmt"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"
)

func main() {
	cli, err := newCLI()
	if err != nil {
		fatalf("%v", err)
	}

	cli.ExecuteAndExit(context.Background())
}

// newCLI builds the command tree. The Builder attaches every command in one
// pass, so a construction fault is reported once and nothing half-built
// remains. The "run" spelling below is load-bearing: scripts/benchmark-
// regression.sh greps main.go for the run subcommand to guard its noise
// gate's target.
func newCLI() (*cmdguard.CLI[appConfig], error) {
	cli, err := cmdguard.NewCLI[appConfig](
		"cqrs-bench",
		"go-cqrs-lite benchmarking tool",
		appConfig{},
		cmdguard.WithCLIVersion(version()),
		cmdguard.WithCLILong(longDescription()),
	)
	if err != nil {
		return nil, fmt.Errorf("create CLI: %w", err)
	}

	builder := cmdguard.NewBuilder(cli)

	cmdguard.Add(builder, "run", runFlags{}, runHandler,
		cmdguard.WithShort("Run a benchmark against a single backend"),
		cmdguard.WithNoArgs())

	cmdguard.Add(builder, "compare", compareFlags{}, compareHandler,
		cmdguard.WithShort("Compare benchmarks across multiple backends"),
		cmdguard.WithNoArgs())

	cmdguard.Add(builder, "sweep", sweepFlags{}, sweepHandler,
		cmdguard.WithShort("Sweep a parameter across benchmark runs"),
		cmdguard.WithNoArgs())

	cmdguard.Add(builder, "list-phases", cmdguard.NoFlags{}, listPhasesHandler,
		cmdguard.WithShort("List all benchmark phases with descriptions"),
		cmdguard.WithNoArgs())

	cmdguard.Add(builder, "layout", layoutFlags{}, layoutHandler,
		cmdguard.WithShort("Explore layout cost model, a pre-deployment what-if analysis"),
		cmdguard.WithNoArgs())

	if err := builder.Build(); err != nil {
		return nil, fmt.Errorf("register commands: %w", err)
	}

	return cli, nil
}

// longDescription is the root help text. The backend block comes from the
// registry and the profile numbers from benchkit, so neither can drift from
// the code.
func longDescription() string {
	sections := []string{
		"cqrs-bench: go-cqrs-lite benchmarking tool",
		backendHelp(),
		profilesHelp(),
		"Codecs:\n  json    JSON encoding (default)\n  cbor    CBOR encoding (compact binary)",
		"Formats:\n" + strings.Join([]string{
			"  auto       A table for a terminal, text anywhere else (default)",
			"  table      Styled terminal table (via go-output)",
			"  text       Human-readable plain text report",
			"  json       Machine-readable JSON",
			"  csv        CSV export (spreadsheet-friendly)",
			"  tsv        TSV export (tab-separated)",
			"  markdown   Markdown table (run, compare, sweep)",
			"  benchstat  benchstat-compatible lines (run only)",
			"  manifest   Config + environment + result as JSON (run only)",
		}, "\n"),
		"Stores:\n" + strings.Join([]string{
			"  Every run, repeat, sweep point, and soak iteration starts from an empty",
			"  store; warmup writes to a throwaway sibling. Without --dir the store",
			"  lives in a temporary directory that is removed afterwards. With --dir",
			"  it stays at <dir>/.cqrs-bench/<backend>/store for inspection and --replay.",
		}, "\n"),
		"Examples:\n" + strings.Join([]string{
			"  cqrs-bench run --backend sqlite --profile dev",
			`  cqrs-bench run --backend sqlite --dsn ":memory:" --profile dev`,
			"  cqrs-bench run --backend pebble --dir /tmp/bench --profile small --codec cbor",
			"  cqrs-bench run --backend memory --profile small --repeat 5",
			"  cqrs-bench run --backend memory --profile dev --soak 5m",
			"  cqrs-bench run --backend memory --profile small --format csv --output results.csv",
			"  cqrs-bench compare --profile small --format markdown",
			"  cqrs-bench compare --profile small --backends sqlite,pebble --dir /tmp/bench",
			"  cqrs-bench sweep --param workers --values 1,2,4,8 --backend memory --profile dev",
		}, "\n"),
	}

	return strings.Join(sections, "\n\n")
}

// profilesHelp is the "Profiles:" block of the long description.
func profilesHelp() string {
	lines := []string{"Profiles:"}

	for _, name := range profileNames() {
		profile, _ := profileByName(name)
		lines = append(lines, fmt.Sprintf("  %-13s %s streams x %s events = %s events, %d goroutines",
			profile.Name, fmtInt(profile.Streams), fmtInt(profile.EventsPerStream),
			fmtInt(profile.TotalEvents()), profile.Concurrency))
	}

	return strings.Join(lines, "\n")
}
