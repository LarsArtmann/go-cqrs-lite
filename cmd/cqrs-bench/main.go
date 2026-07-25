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
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "compare":
		compareCmd(os.Args[2:])
	case "sweep":
		sweepCmd(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println("cqrs-bench version " + version())
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `cqrs-bench — go-cqrs-lite benchmarking tool

Usage:
  cqrs-bench run      --backend <name> [--dsn <dsn>] --profile <name> [flags]
  cqrs-bench compare  --profile <name> [--backends mem,sq,peb] [flags]
  cqrs-bench sweep    --param <name> --values 1,2,4 --backend <name> [flags]

Backends:
  memory    In-memory store (no persistence)
  sqlite    SQLite database (pure-Go, no CGo)
  pebble    PebbleDB LSM-tree store
  postgres  PostgreSQL database (requires --dsn)

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
  text       Human-readable report (default)
  json       Machine-readable JSON
  markdown   Markdown comparison table
  benchstat  benchstat-compatible lines (pipe to benchstat)
  manifest   Config + environment + result as JSON

Examples:
  cqrs-bench run --backend sqlite --dsn ":memory:" --profile dev
  cqrs-bench compare --profile small --format markdown
  cqrs-bench run --backend pebble --dir /tmp/bench --profile small --codec cbor
  cqrs-bench run --backend memory --profile small --repeat 5
  cqrs-bench sweep --param workers --values 1,2,4,8 --backend memory --profile dev
  cqrs-bench sweep --param batchSize --values 1,5,10 --backend sqlite --profile small`)
}

// ── run subcommand ──

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)

	bf := registerBenchFlags(fs)

	warmup := fs.Int("warmup", 0, "Number of warmup operations")
	recovery := fs.Bool("recovery", false, "Enable crash-recovery phase (close, reopen, reload)")
	replay := fs.Bool(
		"replay",
		false,
		"Replay existing store (skip writes, discover streams from journal)",
	)
	cpuprofile := fs.String("cpuprofile", "", "Write CPU profile to file")
	memprofile := fs.String("memprofile", "", "Write heap profile to file")
	_ = fs.Parse(args)

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fatalf("create cpu profile: %v", err)
		}

		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			fatalf("start cpu profile: %v", err)
		}

		defer pprof.StopCPUProfile()
	}

	if *memprofile != "" {
		defer func() {
			f, err := os.Create(*memprofile)
			if err != nil {
				fatalf("create mem profile: %v", err)
			}

			defer f.Close()

			_ = pprof.WriteHeapProfile(f)
		}()
	}

	profile, codec := loadProfileAndCodec(*bf.profileName, *bf.codecName)

	factory, diskPath, cleanup := makeFactory(*bf.backend, *bf.dsn, *bf.dir)
	if cleanup != nil {
		defer cleanup()
	}

	config := benchkit.Config{
		Profile:     profile,
		PayloadSize: *bf.payloadSize,
		Codec:       codec,
		Warmup:      *warmup,
		Repeat:      *bf.repeat,
		Recovery:    *recovery,
		ReplayOnly:  *replay,
		SkipRawSink: *bf.skipRawSink,
		Backend:     *bf.backend,
		DiskPath:    diskPath,
	}

	if sizes, err := parsePayloadSizes(*bf.payloadSizes); err != nil {
		fatalf("invalid --payload-sizes: %v", err)
	} else if len(sizes) > 0 {
		config.PayloadSizes = sizes
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := benchkit.Run(ctx, config, factory)
	if err != nil {
		fatalf("benchmark failed: %v", err)
	}

	writeResult(*bf.format, *bf.output, config, result)
}

// ── compare subcommand ──

func compareCmd(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)

	bf := registerBenchFlags(fs)

	backendList := fs.String("backends", "memory,sqlite,pebble",
		"Comma-separated backend list (memory,sqlite,pebble)")
	_ = fs.Parse(args)

	profile, codec := loadProfileAndCodec(*bf.profileName, *bf.codecName)

	names := strings.Split(*backendList, ",")
	factories := make(map[string]benchkit.Factory, len(names))
	diskPaths := make(map[string]string, len(names))

	for _, name := range names {
		name = strings.TrimSpace(name)

		factory, diskPath, cleanup := makeFactory(name, "", "")
		factories[name] = factory
		diskPaths[name] = diskPath

		if cleanup != nil {
			defer cleanup()
		}
	}

	config := benchkit.Config{
		Profile:     profile,
		PayloadSize: *bf.payloadSize,
		Codec:       codec,
		Repeat:      *bf.repeat,
	}

	if sizes, err := parsePayloadSizes(*bf.payloadSizes); err != nil {
		fatalf("invalid --payload-sizes: %v", err)
	} else if len(sizes) > 0 {
		config.PayloadSizes = sizes
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	results := compareWithDiskPaths(ctx, config, factories, diskPaths)

	writeComparison(*bf.format, *bf.output, results)
}

// ── sweep subcommand ──

func sweepCmd(args []string) {
	fs := flag.NewFlagSet("sweep", flag.ExitOnError)

	bf := registerBenchFlags(fs)

	param := fs.String(
		"param",
		"workers",
		"Parameter to sweep: workers, batchSize, streamLength, gomaxprocs",
	)
	valuesStr := fs.String("values", "1,2,4", "Comma-separated sweep values (e.g. 1,2,4,8)")
	_ = fs.Parse(args)

	profile, codec := loadProfileAndCodec(*bf.profileName, *bf.codecName)

	values, err := parsePayloadSizes(*valuesStr)
	if err != nil {
		fatalf("invalid --values: %v", err)
	}

	if len(values) < 2 {
		fatalf("provide at least 2 values to sweep, got %d", len(values))
	}

	factory, diskPath, cleanup := makeFactory(*bf.backend, *bf.dsn, *bf.dir)
	if cleanup != nil {
		defer cleanup()
	}

	config := benchkit.Config{
		Profile:     profile,
		PayloadSize: *bf.payloadSize,
		Codec:       codec,
		SkipRawSink: *bf.skipRawSink,
		Backend:     *bf.backend,
		DiskPath:    diskPath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var results []benchkit.SweepResult

	switch *param {
	case "workers":
		results = benchkit.WorkerSweep(ctx, config, factory, values)
	case "batchSize", "batch-size", "batch":
		results = benchkit.BatchSizeSweep(ctx, config, factory, values)
	case "streamLength", "stream-length", "stream":
		results = benchkit.StreamLengthSweep(ctx, config, factory, values)
	case "gomaxprocs", "gomax":
		results = benchkit.GOMAXPROCSSweep(ctx, config, factory, values)
	default:
		fatalf("unknown parameter: %s (use workers, batchSize, streamLength, gomaxprocs)", *param)
	}

	writeSweep(*bf.format, *bf.output, results)
}
