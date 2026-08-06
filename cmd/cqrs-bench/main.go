// Command cqrs-bench runs go-cqrs-lite benchmarks against any supported
// backend. It is the CLI front-end for the benchkit library.
//
// Usage:
//
//	cqrs-bench run --backend sqlite --dsn ":memory:" --profile dev
//	cqrs-bench compare --profile small --format markdown
//
// The tool never hardcodes a backend — each backend is a Factory function
// from its stack preset. Adding a new backend means adding one case to the
// factory switch.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

type AppConfig struct {
	cmdguard.Config
}

const longDesc = `cqrs-bench — go-cqrs-lite benchmarking tool

Backends:
  memory     In-memory store (no persistence)
  sqlite     SQLite database (pure-Go modernc driver, optimized pragmas)
  sqlite-cgo SQLite database (CGo mattn driver — 3-5x faster, requires gcc)
  pebble     PebbleDB LSM-tree store
  bbolt      bbolt B+tree store (pure Go, single-writer, etcd-backed)
  postgres   PostgreSQL database (requires --dsn)
  mysql      MySQL/MariaDB database (requires --dsn)
  turso      Turso embedded database (libSQL/SQLite fork)

Profiles:
  dev         100 streams x 5 events     (500 events, 1 goroutine)
  small       1K streams x 10 events     (10K events, 4 goroutines)
  medium      10K streams x 50 events    (500K events, 16 goroutines)
  large       100K streams x 100 events  (10M events, 32 goroutines)
  stress      10K streams x 500 events   (5M events, 64 goroutines)
  write-heavy 10K streams x 100 events   (1M events, 32 goroutines, 90% writes)
  read-heavy  10K streams x 100 events   (1M events, 32 goroutines, 80% reads)
  analytical  10K streams x 10 events    (100K events, 16 goroutines, 90% reads, 5x journal scans)

Codecs:
  json    JSON encoding (default)
  cbor    CBOR encoding (compact binary)

Formats:
  auto       Auto-detect: styled table in terminal, plain text when piped (default)
  table      Styled terminal table with borders and color (via go-output)
  text       Human-readable plain text report
  json       Machine-readable JSON
  csv        CSV export (spreadsheet-friendly)
  tsv        TSV export (tab-separated)
  markdown   Markdown comparison table
  benchstat  benchstat-compatible lines (pipe to benchstat)
  manifest   Config + environment + result as JSON

Examples:
  cqrs-bench run --backend sqlite --dsn ":memory:" --profile dev
  cqrs-bench run --backend memory --profile dev --format table
  cqrs-bench run --backend pebble --dir /tmp/bench --profile small --codec cbor
  cqrs-bench run --backend memory --profile small --repeat 5
  cqrs-bench run --backend memory --profile dev --soak 5m
  cqrs-bench run --backend sqlite --dsn ":memory:" --profile dev --skip-snapshot
  cqrs-bench run --backend memory --profile small --format csv --output results.csv
  cqrs-bench compare --profile small --format table
  cqrs-bench compare --profile small --format markdown
  cqrs-bench compare --profile small --format csv
  cqrs-bench sweep --param workers --values 1,2,4,8 --backend memory --profile dev
  cqrs-bench sweep --param batchSize --values 1,5,10 --backend sqlite --profile small`

func main() {
	cli, err := cmdguard.NewCLI[AppConfig](
		"cqrs-bench",
		"go-cqrs-lite benchmarking tool",
		AppConfig{},
		cmdguard.WithCLIVersion(version()),
		cmdguard.WithCLILong(longDesc),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI: %v\n", err)
		os.Exit(1)
	}

	runCmd, err := cmdguard.NewCommand[AppConfig, *RunFlags]("run", &RunFlags{},
		runHandler,
		cmdguard.WithShort("Run a benchmark against a single backend"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating run command: %v\n", err)
		os.Exit(1)
	}

	compareCmd, err := cmdguard.NewCommand[AppConfig, *CompareFlags]("compare", &CompareFlags{},
		compareHandler,
		cmdguard.WithShort("Compare benchmarks across multiple backends"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating compare command: %v\n", err)
		os.Exit(1)
	}

	sweepCmd, err := cmdguard.NewCommand[AppConfig, *SweepFlags]("sweep", &SweepFlags{},
		sweepHandler,
		cmdguard.WithShort("Sweep a parameter across benchmark runs"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating sweep command: %v\n", err)
		os.Exit(1)
	}

	if err := cmdguard.AddCommand(cli, runCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding run command: %v\n", err)
		os.Exit(1)
	}

	if err := cmdguard.AddCommand(cli, compareCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding compare command: %v\n", err)
		os.Exit(1)
	}

	if err := cmdguard.AddCommand(cli, sweepCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding sweep command: %v\n", err)
		os.Exit(1)
	}

	cli.ExecuteAndExit(context.Background())
}

// ── run subcommand ──

func runHandler(ctx context.Context, _ *AppConfig, flags *RunFlags) error {
	if flags.CPUProfile != "" {
		f, err := os.Create(flags.CPUProfile)
		if err != nil {
			fatalf("create cpu profile: %v", err)
		}

		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			fatalf("start cpu profile: %v", err)
		}

		defer pprof.StopCPUProfile()
	}

	if flags.MemProfile != "" {
		defer func() {
			f, err := os.Create(flags.MemProfile)
			if err != nil {
				fatalf("create mem profile: %v", err)
			}

			defer f.Close()

			_ = pprof.WriteHeapProfile(f)
		}()
	}

	profile, codec := loadProfileAndCodec(flags.Profile, flags.Codec)

	factory, diskPath, cleanup := makeFactory(flags.Backend, flags.DSN, flags.Dir, flags.Durability)
	if cleanup != nil {
		defer cleanup()
	}

	soak := flags.Soak.Duration()

	config := benchkit.Config{
		Profile:      profile,
		PayloadSize:  flags.PayloadSize,
		Codec:        codec,
		Warmup:       flags.Warmup,
		Repeat:       flags.Repeat,
		Recovery:     flags.Recovery,
		ReplayOnly:   flags.Replay,
		SkipRawSink:  flags.SkipRawSink,
		SkipJourney:  flags.SkipJourney,
		SkipQuery:    flags.SkipQuery,
		SkipSnapshot: flags.SkipSnapshot,
		SkipMixed:    flags.SkipMixed,
		Backend:      flags.Backend,
		DiskPath:     diskPath,
	}
	applyProgress(&config, flags.Progress.Duration(), flags.Quiet)

	if sizes, err := parsePayloadSizes(flags.PayloadSizes); err != nil {
		fatalf("invalid --payload-sizes: %v", err)
	} else if len(sizes) > 0 {
		config.PayloadSizes = sizes
	}

	runCtx, cancel := context.WithTimeout(ctx, max(30*time.Minute, soak*2))
	defer cancel()

	if soak > 0 {
		var soakProgressWriter io.Writer = os.Stderr
		if flags.Quiet {
			soakProgressWriter = nil
		}

		soakResult, err := benchkit.RunSoak(runCtx, benchkit.SoakConfig{
			Duration:       soak,
			ReportInterval: 10 * time.Second,
			ProgressWriter: soakProgressWriter,
			Config:         config,
		}, factory)
		if err != nil {
			fatalf("soak test failed: %v", err)
		}

		writeSoakResult(flags.Format, flags.Output, soakResult)

		return nil
	}

	result, err := benchkit.Run(runCtx, config, factory)
	if err != nil {
		fatalf("benchmark failed: %v", err)
	}

	writeResult(flags.Format, flags.Output, config, result)

	return nil
}

// ── compare subcommand ──

func compareHandler(ctx context.Context, _ *AppConfig, flags *CompareFlags) error {
	profile, codec := loadProfileAndCodec(flags.Profile, flags.Codec)

	names := strings.Split(flags.Backends, ",")
	factories := make(map[string]benchkit.Factory, len(names))
	diskPaths := make(map[string]string, len(names))

	for _, name := range names {
		name = strings.TrimSpace(name)

		factory, diskPath, cleanup := makeFactory(name, "", "", flags.Durability)
		factories[name] = factory
		diskPaths[name] = diskPath

		if cleanup != nil {
			defer cleanup()
		}
	}

	config := benchkit.Config{
		Profile:      profile,
		PayloadSize:  flags.PayloadSize,
		Codec:        codec,
		Repeat:       flags.Repeat,
		SkipRawSink:  flags.SkipRawSink,
		SkipJourney:  flags.SkipJourney,
		SkipQuery:    flags.SkipQuery,
		SkipSnapshot: flags.SkipSnapshot,
	}
	applyProgress(&config, flags.Progress.Duration(), flags.Quiet)

	if sizes, err := parsePayloadSizes(flags.PayloadSizes); err != nil {
		fatalf("invalid --payload-sizes: %v", err)
	} else if len(sizes) > 0 {
		config.PayloadSizes = sizes
	}

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	results := compareWithDiskPaths(runCtx, config, factories, diskPaths)

	writeComparison(flags.Format, flags.Output, results)

	return nil
}

// ── sweep subcommand ──

func sweepHandler(ctx context.Context, _ *AppConfig, flags *SweepFlags) error {
	profile, codec := loadProfileAndCodec(flags.Profile, flags.Codec)

	values, err := parsePayloadSizes(flags.Values)
	if err != nil {
		fatalf("invalid --values: %v", err)
	}

	if len(values) < 2 {
		fatalf("provide at least 2 values to sweep, got %d", len(values))
	}

	factory, diskPath, cleanup := makeFactory(flags.Backend, flags.DSN, flags.Dir, flags.Durability)
	if cleanup != nil {
		defer cleanup()
	}

	config := benchkit.Config{
		Profile:      profile,
		PayloadSize:  flags.PayloadSize,
		Codec:        codec,
		SkipRawSink:  flags.SkipRawSink,
		SkipJourney:  flags.SkipJourney,
		SkipQuery:    flags.SkipQuery,
		SkipSnapshot: flags.SkipSnapshot,
		Backend:      flags.Backend,
		DiskPath:     diskPath,
	}

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	var results []benchkit.SweepResult

	switch flags.Param {
	case "workers":
		results = benchkit.WorkerSweep(runCtx, config, factory, values)
	case "batchSize", "batch-size", "batch":
		results = benchkit.BatchSizeSweep(runCtx, config, factory, values)
	case "streamLength", "stream-length", "stream":
		results = benchkit.StreamLengthSweep(runCtx, config, factory, values)
	case "gomaxprocs", "gomax":
		results = benchkit.GOMAXPROCSSweep(runCtx, config, factory, values)
	default:
		fatalf(
			"unknown parameter: %s (use workers, batchSize, streamLength, gomaxprocs)",
			flags.Param,
		)
	}

	writeSweep(flags.Format, flags.Output, results)

	return nil
}
