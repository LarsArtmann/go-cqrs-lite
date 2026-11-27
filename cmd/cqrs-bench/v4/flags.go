package main

import (
	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"
)

// appConfig is the root configuration. cqrs-bench has no root flags of its
// own; cmdguard's config-file flags hang off it.
type appConfig struct {
	cmdguard.Config
}

// BenchFlags are the flags every benchmark command shares. run, compare,
// and sweep all build the same benchkit.Config from them (BenchFlags.config),
// so a flag cannot reach one command and be dropped by another.
type BenchFlags struct {
	Backend        string            `default:"memory" flag:"backend"                  help:"Backend to benchmark: memory, system-memory, sqlite, sqlite-cgo, bbolt, pebble, turso, duckdb, postgres, mysql (aliases: mem, sys-mem, sq, sq3, bolt, peb, duck, pg, maria, mariadb)"`
	DSN            string            `default:""       flag:"dsn"                      help:"Database connection string: required by postgres and mysql; for sqlite and duckdb a path names the store file, which a measuring run needs absent or empty"`
	Dir            string            `default:""       flag:"dir"                      help:"Directory for an embedded backend's store (sqlite, sqlite-cgo, bbolt, pebble, turso, duckdb); created when missing and kept under <dir>/.cqrs-bench/<backend>/ for --replay"`
	Profile        string            `default:"dev"    flag:"profile"                  help:"Workload profile"                                                                                                                                                                     values:"dev,small,medium,large,stress,write-heavy,read-heavy,analytical"`
	Codec          string            `default:"json"   flag:"codec"                    help:"Payload codec"                                                                                                                                                                        values:"json,cbor"`
	Format         string            `default:"auto"   flag:"format"                   help:"Output format (benchstat and manifest: run only; auto renders a table to a terminal and text anywhere else)"                                                                          values:"auto,table,text,json,csv,tsv,markdown,benchstat,manifest"`
	Output         string            `default:""       flag:"output"                   help:"Output file (default: stdout); files are replaced atomically and never contain color codes"`
	PayloadSize    int               `default:"256"    flag:"payload-size"             help:"Payload size in bytes per event"`
	PayloadSizes   string            `default:""       flag:"payload-sizes"            help:"Comma-separated payload sizes (at least two) for a MIXED workload, e.g. 64,256,4096; overrides --payload-size"`
	Durability     string            `default:""       flag:"durability"               help:"Durability tier (default: normal; ignored by memory, system-memory, duckdb, and mysql)"                                                                                               values:"strict,normal,relaxed"`
	SkipReads      bool              `default:"false"  flag:"skip-reads"               help:"Skip the read and versioned-read phases (stream loads and journal scans)"`
	SkipReadModels bool              `default:"false"  flag:"skip-read-models"         help:"Skip the read-model phase (KV store Set and Get)"`
	SkipProjection bool              `default:"false"  flag:"skip-projections"         help:"Skip the projection and checkpoint phases"`
	SkipMetaEngine bool              `default:"false"  flag:"skip-metaengine"          help:"Skip the metaengine planner phase"`
	SkipBatchWrite bool              `default:"false"  flag:"skip-batch-write"         help:"Skip the AppendBatch throughput phase (its extra events inflate the journal count)"`
	SkipRawSink    bool              `default:"false"  flag:"skip-raw-sink"            help:"Skip the raw prebuilt-event sink phase"`
	SkipJourney    bool              `default:"false"  flag:"skip-journey"             help:"Skip the end-to-end publish, projection, and query journey phase"`
	SkipQuery      bool              `default:"false"  flag:"skip-query"               help:"Skip the typed query dispatch phase"`
	SkipSnapshot   bool              `default:"false"  flag:"skip-snapshot"            help:"Skip the snapshot and cache hit-rate phase"`
	SkipMixed      bool              `default:"false"  flag:"skip-mixed"               help:"Skip the concurrent read-during-write phase"`
	Progress       cmdguard.Duration `default:"5s"     flag:"progress"                 help:"Progress update interval to stderr (0 disables)"`
	Quiet          bool              `default:"false"  flag:"quiet"                    help:"Suppress all progress output (stderr). Result only. Implies --progress=0"`
	Repeat         int               `default:"0"      flag:"repeat"                   help:"Run N times, report median (reduces ~20% variance)"`
	Strict         bool              `default:"false"  flag:"strict"                   help:"CI gate: fail when a phase the backend can run is skipped, on integrity errors, and (with --repeat 2 or more) when a headline metric's cross-run CoV is NOISY"`
	Timeout        cmdguard.Duration `default:"30m"    flag:"timeout"                  help:"Wall-clock budget per run (every repeat, compared backend, and sweep point is a run); 0 disables the budget"`
	Interpolated   bool              `default:"false"  flag:"interpolated-percentiles" help:"Interpolate P50-P99 between samples (smoother percentiles for small-n runs; default nearest-rank)"`
	ReservoirSize  int               `default:"0"      flag:"reservoir-size"           help:"Latency samples each phase retains once the workload exceeds it (default 10000). Raise for 10M+ event profiles; Min/Max/Mean stay exact at any size"`
}

// runFlags extends BenchFlags with the flags of the run command.
type runFlags struct {
	BenchFlags

	Warmup      int               `default:"0"     flag:"warmup"       help:"Warmup operations before timing, written to a separate throwaway store"`
	Recovery    bool              `default:"false" flag:"recovery"     help:"Enable the crash-recovery phase (close, reopen, reload)"`
	Replay      bool              `default:"false" flag:"replay"       help:"Replay an existing store (skip writes, discover streams from the journal); needs --dir or a file --dsn"`
	CPUProfile  string            `default:""      flag:"cpuprofile"   help:"Write a CPU profile to file"`
	MemProfile  string            `default:""      flag:"memprofile"   help:"Write a heap profile to file"`
	Soak        cmdguard.Duration `default:"0"     flag:"soak"         help:"Run in soak mode for the given duration (e.g. 5m, 1h). Repeats the workload on a fresh store and reports leak and degradation trends"`
	IncludeRuns bool              `default:"false" flag:"include-runs" help:"With --format manifest: serialize every repeat run (runs[]) alongside the median; the file grows N-fold"`
}

// compareFlags extends BenchFlags with the flags of the compare command.
type compareFlags struct {
	BenchFlags

	Backends string `default:"memory,sqlite,pebble" flag:"backends" help:"Comma-separated backends to compare, in report order"`
}

// sweepFlags extends BenchFlags with the flags of the sweep command.
type sweepFlags struct {
	BenchFlags

	Param  string `default:"workers" flag:"param"  help:"Parameter to sweep: workers, batchSize, streamLength, gomaxprocs"`
	Values string `default:""        flag:"values" help:"Comma-separated values (at least two) for the parameter, e.g. 1,2,4,8"`
}

// layoutFlags are the flags of the layout command: pre-deployment "what if"
// exploration of the metaengine layout cost model.
type layoutFlags struct {
	Priority string `default:""      flag:"priority" help:"Filter to a single priority: balanced, read-speed, write-speed, storage-space (default: show all)"`
	Layout   string `default:""      flag:"layout"   help:"Filter to a single storage layout: kv, lsm, row, columnar (default: show all)"`
	Format   string `default:"auto"  flag:"format"   help:"Output format"                                                                                     values:"auto,table,text,json"`
	Output   string `default:""      flag:"output"   help:"Output file (default: stdout)"`
	Verbose  bool   `default:"false" flag:"verbose"  help:"Show the cost breakdown (ReadCost, WriteCost, StorageCost) for each layout option"`
}
