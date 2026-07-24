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
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
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

	backend := fs.String("backend", "memory", "Backend: memory, sqlite, pebble")
	dsn := fs.String("dsn", "", "Database connection string (sqlite)")
	dir := fs.String("dir", "", "Database directory (pebble)")
	profileName := fs.String("profile", "dev", "Workload profile")
	codecName := fs.String("codec", "json", "Payload codec: json, cbor")
	format := fs.String("format", "text", "Output format: text, json, benchstat, manifest")
	output := fs.String("output", "", "Output file (default: stdout)")
	payloadSize := fs.Int("payload-size", 256, "Payload size in bytes per event")
	payloadSizes := fs.String(
		"payload-sizes",
		"",
		"Comma-separated payload sizes for a MIXED workload (e.g. 64,256,4096). Overrides --payload-size",
	)
	warmup := fs.Int("warmup", 0, "Number of warmup operations")
	repeat := fs.Int("repeat", 0, "Run N times, report median (reduces ~20% variance)")
	skipRawSink := fs.Bool("skip-raw-sink", false, "Skip raw prebuilt-event sink phase")
	recovery := fs.Bool("recovery", false, "Enable crash-recovery phase (close, reopen, reload)")
	replay := fs.Bool(
		"replay",
		false,
		"Replay existing store (skip writes, discover streams from journal)",
	)
	_ = fs.Parse(args)

	profile, ok := benchkit.ProfileByName(*profileName)
	if !ok {
		fatalf("unknown profile: %s", *profileName)
	}

	codec := parseCodec(*codecName)

	factory, diskPath, cleanup := makeFactory(*backend, *dsn, *dir)
	if cleanup != nil {
		defer cleanup()
	}

	config := benchkit.Config{
		Profile:     profile,
		PayloadSize: *payloadSize,
		Codec:       codec,
		Warmup:      *warmup,
		Repeat:      *repeat,
		Recovery:    *recovery,
		ReplayOnly:  *replay,
		SkipRawSink: *skipRawSink,
		Backend:     *backend,
		DiskPath:    diskPath,
	}

	if sizes, err := parsePayloadSizes(*payloadSizes); err != nil {
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

	writeResult(*format, *output, config, result)
}

// ── compare subcommand ──

func compareCmd(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)

	profileName := fs.String("profile", "dev", "Workload profile")
	codecName := fs.String("codec", "json", "Payload codec: json, cbor")
	format := fs.String("format", "text", "Output format: text, json, markdown")
	output := fs.String("output", "", "Output file (default: stdout)")
	backendList := fs.String("backends", "memory,sqlite,pebble",
		"Comma-separated backend list (memory,sqlite,pebble)")
	payloadSize := fs.Int("payload-size", 256, "Payload size in bytes per event")
	payloadSizes := fs.String(
		"payload-sizes",
		"",
		"Comma-separated payload sizes for a MIXED workload (e.g. 64,256,4096). Overrides --payload-size",
	)
	repeat := fs.Int("repeat", 0, "Run N times per backend, report median (reduces ~20% variance)")
	_ = fs.Parse(args)

	profile, ok := benchkit.ProfileByName(*profileName)
	if !ok {
		fatalf("unknown profile: %s", *profileName)
	}

	codec := parseCodec(*codecName)

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
		PayloadSize: *payloadSize,
		Codec:       codec,
		Repeat:      *repeat,
	}

	if sizes, err := parsePayloadSizes(*payloadSizes); err != nil {
		fatalf("invalid --payload-sizes: %v", err)
	} else if len(sizes) > 0 {
		config.PayloadSizes = sizes
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	results := compareWithDiskPaths(ctx, config, factories, diskPaths)

	writeComparison(*format, *output, results)
}

// ── sweep subcommand ──

func sweepCmd(args []string) {
	fs := flag.NewFlagSet("sweep", flag.ExitOnError)

	param := fs.String(
		"param",
		"workers",
		"Parameter to sweep: workers, batchSize, streamLength, gomaxprocs",
	)
	valuesStr := fs.String("values", "1,2,4", "Comma-separated sweep values (e.g. 1,2,4,8)")
	backend := fs.String("backend", "memory", "Backend: memory, sqlite, pebble")
	dsn := fs.String("dsn", "", "Database connection string (sqlite)")
	dir := fs.String("dir", "", "Database directory (pebble)")
	profileName := fs.String("profile", "dev", "Workload profile")
	codecName := fs.String("codec", "json", "Payload codec: json, cbor")
	format := fs.String("format", "text", "Output format: text, json, benchstat, manifest")
	output := fs.String("output", "", "Output file (default: stdout)")
	payloadSize := fs.Int("payload-size", 256, "Payload size in bytes per event")
	skipRawSink := fs.Bool("skip-raw-sink", false, "Skip raw prebuilt-event sink phase")
	_ = fs.Parse(args)

	profile, ok := benchkit.ProfileByName(*profileName)
	if !ok {
		fatalf("unknown profile: %s", *profileName)
	}

	codec := parseCodec(*codecName)

	values, err := parsePayloadSizes(*valuesStr)
	if err != nil {
		fatalf("invalid --values: %v", err)
	}

	if len(values) < 2 {
		fatalf("provide at least 2 values to sweep, got %d", len(values))
	}

	factory, diskPath, cleanup := makeFactory(*backend, *dsn, *dir)
	if cleanup != nil {
		defer cleanup()
	}

	config := benchkit.Config{
		Profile:     profile,
		PayloadSize: *payloadSize,
		Codec:       codec,
		SkipRawSink: *skipRawSink,
		Backend:     *backend,
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

	writeSweep(*format, *output, results)
}

// ── factory ──

// compareWithDiskPaths runs the benchmark against each backend, setting
// per-backend DiskPath so disk metrics are populated in comparison tables.
// This mirrors benchkit.Compare but injects the correct DiskPath per backend.
func compareWithDiskPaths(
	ctx context.Context,
	config benchkit.Config,
	factories map[string]benchkit.Factory,
	diskPaths map[string]string,
) map[string]*benchkit.Result {
	results := make(map[string]*benchkit.Result, len(factories))

	for name, factory := range factories {
		cfg := config
		cfg.Backend = name
		cfg.DiskPath = diskPaths[name]

		result, err := benchkit.Run(ctx, cfg, factory)
		if err != nil {
			results[name] = &benchkit.Result{
				Backend:   name,
				Profile:   cfg.Profile.Name,
				Timestamp: time.Now(),
				Error:     err.Error(),
			}

			continue
		}

		results[name] = result
	}

	return results
}

func makeFactory(backend, dsn, dir string) (benchkit.Factory, string, func()) {
	var (
		diskPath string
		cleanup  func()
	)

	switch backend {
	case "memory", "mem":
		return memory.New, "", nil

	case "sqlite", "sq":
		dbDir := dir
		if dbDir == "" {
			dbDir = mkTempDir()
			cleanup = func() { _ = os.RemoveAll(dbDir) }
		}

		dbPath := filepath.Join(dbDir, "bench.db")
		if dsn == "" {
			dsn = dbPath
		}

		diskPath = dbDir

		return func() (*stack.Bundle, error) { return sqlite.New(dsn) }, diskPath, cleanup

	case "pebble", "peb":
		pebDir := dir
		if pebDir == "" {
			pebDir = mkTempDir()
			cleanup = func() { _ = os.RemoveAll(pebDir) }
		}

		diskPath = pebDir

		return func() (*stack.Bundle, error) {
			b, err := pebble.New(pebDir)
			if err != nil {
				return nil, err
			}

			return b.Bundle, nil
		}, diskPath, cleanup

	case "postgres", "pg":
		if dsn == "" {
			fatalf(
				"postgres backend requires --dsn (e.g. postgres://user:pass@localhost:5432/bench?sslmode=disable)",
			)
		}

		return func() (*stack.Bundle, error) { return postgres.New(dsn) }, "", nil

	default:
		fatalf("unknown backend: %s (use memory, sqlite, or pebble)", backend)

		return nil, "", nil // unreachable
	}
}

func mkTempDir() string {
	dir, err := os.MkdirTemp("", "cqrs-bench-*")
	if err != nil {
		fatalf("create temp dir: %v", err)
	}

	return dir
}

func parseCodec(name string) codec.Codec {
	switch name {
	case "json":
		return codec.JSONCodec{}
	case "cbor":
		return codec.CBORCodec{}
	default:
		fatalf("unknown codec: %s (use json or cbor)", name)

		return nil
	}
}

// parsePayloadSizes parses a comma-separated list of payload sizes (e.g.
// "64,256,4096") into an int slice. Returns nil for an empty string (meaning:
// use the single --payload-size). Returns an error on malformed input.
func parsePayloadSizes(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	sizes := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid size %q: %w", part, err)
		}

		if n <= 0 {
			return nil, fmt.Errorf("size must be > 0, got %d", n)
		}

		sizes = append(sizes, n)
	}

	if len(sizes) < 2 {
		return nil, fmt.Errorf("provide at least 2 sizes for a mixed workload, got %d", len(sizes))
	}

	return sizes, nil
}

// ── output ──

func writeResult(format, output string, config benchkit.Config, result *benchkit.Result) {
	w := openOutput(output)
	defer closeOutput(w)

	switch format {
	case "json":
		if err := benchkit.WriteJSON(w, result); err != nil {
			fatalf("write JSON: %v", err)
		}
	case "benchstat":
		benchkit.WriteBenchstat(w, result)
	case "manifest":
		if err := benchkit.WriteManifest(w, config, result); err != nil {
			fatalf("write manifest: %v", err)
		}
	default:
		benchkit.PrintReport(w, result)
	}
}

func writeComparison(
	format, output string,
	results map[string]*benchkit.Result,
) {
	w := openOutput(output)
	defer closeOutput(w)

	switch format {
	case "json":
		if err := benchkit.WriteComparisonJSON(w, results); err != nil {
			fatalf("write JSON: %v", err)
		}
	case "markdown":
		benchkit.PrintMarkdown(w, results)
	default:
		benchkit.PrintComparison(w, results)
	}
}

func writeSweep(format, output string, results []benchkit.SweepResult) {
	w := openOutput(output)
	defer closeOutput(w)

	switch format {
	case "json":
		if err := benchkit.WriteSweepJSON(w, results); err != nil {
			fatalf("write JSON: %v", err)
		}
	default:
		benchkit.PrintSweep(w, results)
	}
}

func openOutput(path string) *os.File {
	if path == "" || path == "-" {
		return os.Stdout
	}

	f, err := os.Create(path)
	if err != nil {
		fatalf("create output file: %v", err)
	}

	return f
}

func closeOutput(f *os.File) {
	if f != os.Stdout {
		_ = f.Close()
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cqrs-bench: "+format+"\n", args...)
	os.Exit(1)
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}

	v := info.Main.Version
	if v != "" && v != "(devel)" {
		return v
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
			return "(devel, " + setting.Value[:7] + ")"
		}
	}

	return "(devel)"
}
