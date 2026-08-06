package main

import (
	"os"
	"time"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

// BenchFlags holds the shared benchmark flags wired onto every cqrs-bench
// subcommand (run, compare, sweep).
type BenchFlags struct {
	Backend      string            `default:"memory" flag:"backend"       help:"Backend: memory, sqlite, sqlite-cgo, bbolt, pebble, postgres, mysql, duckdb, turso"`
	DSN          string            `default:""       flag:"dsn"           help:"Database connection string (sqlite, postgres, mysql, duckdb)"`
	Dir          string            `default:""       flag:"dir"           help:"Database directory (bbolt, pebble, duckdb)"`
	Profile      string            `default:"dev"    flag:"profile"       help:"Workload profile"`
	Codec        string            `default:"json"   flag:"codec"         help:"Payload codec: json, cbor"`
	Format       string            `default:"auto"   flag:"format"        help:"Output format: auto, table, text, json, csv, tsv, markdown, benchstat, manifest (auto picks table in terminal, text when piped)"`
	Output       string            `default:""       flag:"output"        help:"Output file (default: stdout)"`
	PayloadSize  int               `default:"256"    flag:"payload-size"  help:"Payload size in bytes per event"`
	PayloadSizes string            `default:""       flag:"payload-sizes" help:"Comma-separated payload sizes for a MIXED workload (e.g. 64,256,4096). Overrides --payload-size"`
	Durability   string            `default:""       flag:"durability"    help:"Durability tier: strict, normal, relaxed (default: normal)"`
	SkipBatchWrite bool             `default:"false"  flag:"skip-batch-write" help:"Skip AppendBatch throughput phase (extra events inflate journal count)"`
	SkipRawSink  bool              `default:"false"  flag:"skip-raw-sink" help:"Skip raw prebuilt-event sink phase"`
	SkipJourney  bool              `default:"false"  flag:"skip-journey"  help:"Skip end-to-end publish→projection→query journey phase"`
	SkipQuery    bool              `default:"false"  flag:"skip-query"    help:"Skip typed query dispatch phase"`
	SkipSnapshot bool              `default:"false"  flag:"skip-snapshot" help:"Skip snapshot/cache hit-rate phase"`
	SkipMixed    bool              `default:"false"  flag:"skip-mixed"    help:"Skip concurrent read-during-write phase"`
	Progress     cmdguard.Duration `default:"5s"     flag:"progress"      help:"Progress update interval to stderr (0 disables)"`
	Quiet        bool              `default:"false"  flag:"quiet"         help:"Suppress all progress output (stderr). Result only. Implies --progress=0"`
	Repeat       int               `default:"0"      flag:"repeat"        help:"Run N times, report median (reduces ~20% variance)"`
	Strict       bool              `default:"false"  flag:"strict"        help:"Fail if any phase is skipped (missing bundle component or config flag). For CI gates."`
}

// RunFlags extends BenchFlags with run-specific flags.
type RunFlags struct {
	BenchFlags

	Warmup     int               `default:"0"     flag:"warmup"     help:"Number of warmup operations"`
	Recovery   bool              `default:"false" flag:"recovery"   help:"Enable crash-recovery phase (close, reopen, reload)"`
	Replay     bool              `default:"false" flag:"replay"     help:"Replay existing store (skip writes, discover streams from journal)"`
	CPUProfile string            `default:""      flag:"cpuprofile" help:"Write CPU profile to file"`
	MemProfile string            `default:""      flag:"memprofile" help:"Write heap profile to file"`
	Soak       cmdguard.Duration `default:"0"     flag:"soak"       help:"Run in soak mode for the given duration (e.g. 5m, 1h). Repeats the workload and reports leak/degradation trends"`
}

// CompareFlags extends BenchFlags with compare-specific flags.
type CompareFlags struct {
	BenchFlags

	Backends string `default:"memory,sqlite,pebble" flag:"backends" help:"Comma-separated backend list (memory,sqlite,sqlite-cgo,bbolt,pebble)"`
}

// SweepFlags extends BenchFlags with sweep-specific flags.
type SweepFlags struct {
	BenchFlags

	Param  string `default:"workers" flag:"param"  help:"Parameter to sweep: workers, batchSize, streamLength, gomaxprocs"`
	Values string `default:""        flag:"values" help:"Comma-separated values for the parameter (e.g. 1,2,4,8)"`
}

// ListPhasesFlags holds the (empty) flags for the list-phases subcommand.
type ListPhasesFlags struct{}

// loadProfileAndCodec resolves the workload profile and payload codec from
// their flag values, calling fatalf on an unknown profile name. Used by every
// subcommand that actually runs a benchmark.
func loadProfileAndCodec(profileName, codecName string) (benchkit.Profile, codec.Codec) {
	profile, ok := benchkit.ProfileByName(profileName)
	if !ok {
		fatalf("unknown profile: %s", profileName)
	}

	return profile, parseCodec(codecName)
}

// applyProgress sets the progress writer and interval on the benchmark config
// when the interval is non-zero. --quiet overrides to suppress all progress.
func applyProgress(config *benchkit.Config, interval time.Duration, quiet bool) {
	if quiet {
		return
	}

	if interval > 0 {
		config.ProgressWriter = os.Stderr
		config.ProgressInterval = interval
	}
}
