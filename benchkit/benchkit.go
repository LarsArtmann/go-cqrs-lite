// Package benchkit provides a factory-driven benchmarking suite for
// [stack.Bundle] presets — the performance equivalent of
// [stack/contracttest].
//
// A deployer provides a [Factory] (a function that returns a fresh
// *stack.Bundle), and benchkit runs a suite of realistic write, read,
// read-model, and projection workloads while collecting latency percentiles,
// throughput, memory deltas, and storage footprint.
//
// # Quick start
//
//	result, err := benchkit.Run(ctx, benchkit.Config{
//	    Profile:     benchkit.ProfileDev,
//	    PayloadSize: 256,
//	}, func() (*stack.Bundle, error) {
//	    return sqlite.New(filepath.Join(dir, "bench.db"))
//	})
//	if err != nil { ... }
//	benchkit.PrintReport(os.Stdout, result)
//
// # Cross-backend comparison
//
//	results, err := benchkit.Compare(ctx, config, map[string]benchkit.Factory{
//	    "memory": func() (*stack.Bundle, error) { return memory.New() },
//	    "sqlite": func() (*stack.Bundle, error) { return sqlite.New(":memory:") },
//	})
//	benchkit.PrintComparison(os.Stdout, results)
//
// The tool never imports a backend driver. Switching backends is a one-line
// factory change, mirroring the library's deployer-first philosophy.
package benchkit

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// Factory creates a fresh [stack.Bundle] for benchmarking. Each call should
// produce an independent Bundle with its own backing resources (temp directory,
// etc.). The caller is responsible for calling Bundle.Close after the run.
type Factory func() (*stack.Bundle, error)

// Config defines a benchmark run.
type Config struct {
	// Profile controls the scale: number of aggregates, events per aggregate,
	// concurrency level, read/write ratio, and batch size.
	// Use a named [Profile] or customize the fields directly.
	Profile Profile

	// PayloadSize controls the synthetic payload byte size per event.
	// Default: 256 bytes.
	PayloadSize int

	// Codec controls payload encoding for event creation.
	// nil defaults to [codec.JSONCodec].
	Codec codec.Codec

	// Duration caps the wall-clock time. Zero means run to completion.
	Duration time.Duration

	// Warmup runs this many write+load cycles before measurement begins
	// to warm caches and JIT compilation. Zero disables warmup.
	Warmup int

	// Concurrency overrides Profile.Concurrency. Zero means use the profile.
	Concurrency int

	// Seed controls the deterministic random data generator.
	// Same seed = same data across runs. Default: 1.
	Seed int64

	// Backend is a human-readable label for the result (e.g. "sqlite").
	// When empty, Run sets it to "unknown".
	Backend string

	// DiskPath is the filesystem path to measure for disk footprint.
	// If set, the runner walks this path after the workload to report
	// on-disk database size. If empty, disk metrics are zero.
	DiskPath string
}

// Result is the output of a single benchmark run against one backend.
type Result struct {
	// Identification
	Backend   string        `json:"backend"`
	Profile   string        `json:"profile"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`

	// Workload
	Streams         int `json:"aggregates"`
	EventsPerStream int `json:"eventsPerAggregate"`
	TotalEvents     int `json:"totalEvents"`
	PayloadBytes    int `json:"payloadBytesPerEvent"`

	// Write metrics
	WriteLatency    LatencyStats `json:"writeLatency"`
	WriteThroughput float64      `json:"writeThroughput"`

	// Read metrics
	LoadLatency  LatencyStats  `json:"loadLatency"`
	ReadAllTime  time.Duration `json:"readAllTime"`
	ReadFromTime time.Duration `json:"readFromTime"`

	// Read model metrics (raw kv.Store Set/Get)
	ReadModelGet LatencyStats `json:"readModelGet"`
	ReadModelSet LatencyStats `json:"readModelSet"`

	// Projection metrics (zero-valued when no projections ran)
	ProjectionLag    time.Duration `json:"projectionLag"`
	ProjectionEvents int64         `json:"projectionEvents"`

	// Resource metrics
	Memory ResourceStats `json:"memory"`
	CPU    ResourceStats `json:"cpu"`
	Disk   DiskStats     `json:"disk"`

	// Codec used for event payloads
	Codec string `json:"codec"`

	// Error captures a non-fatal error that prevented a phase from completing
	// (e.g. backend doesn't support SeekableJournal). The run still succeeds;
	// the affected metrics are zero-valued.
	Error string `json:"error,omitempty"`
}

// LatencyStats holds percentile latency data.
type LatencyStats struct {
	Count int64         `json:"count"`
	P50   time.Duration `json:"p50"`
	P75   time.Duration `json:"p75"`
	P90   time.Duration `json:"p90"`
	P95   time.Duration `json:"p95"`
	P99   time.Duration `json:"p99"`
	P100  time.Duration `json:"p100"`
	Mean  time.Duration `json:"mean"`
}

// ResourceStats holds resource usage snapshots.
type ResourceStats struct {
	Before uint64 `json:"before"`
	After  uint64 `json:"after"`
	Delta  uint64 `json:"delta"`
}

// DiskStats holds storage footprint data.
type DiskStats struct {
	DatabaseBytes int64   `json:"databaseBytes"`
	EventBytes    int64   `json:"eventBytes"`
	OverheadBytes int64   `json:"overheadBytes"`
	OverheadPct   float64 `json:"overheadPct"`
}

// Run executes a benchmark against one backend and returns the result.
//
// The factory is called once to create the Bundle. All phases (write, read,
// read-model, projection, durability) run against that same Bundle. The Bundle
// is closed automatically after the run.
func Run(ctx context.Context, config Config, factory Factory) (*Result, error) {
	return newRunner(config, factory).run(ctx)
}

// Compare executes the same benchmark against multiple backends and returns
// a map of backend name to Result. Each backend gets a fresh Bundle.
//
// Backends whose factory returns an error are included in the result map with
// a zero-valued Result containing the error message — they do not abort the
// comparison.
func Compare(
	ctx context.Context,
	config Config,
	factories map[string]Factory,
) (map[string]*Result, error) {
	results := make(map[string]*Result, len(factories))

	for name, factory := range factories {
		cfg := config
		cfg.Backend = name

		result, err := Run(ctx, cfg, factory)
		if err != nil {
			results[name] = &Result{
				Backend:   name,
				Profile:   cfg.Profile.Name,
				Timestamp: time.Now(),
				Error:     err.Error(),
			}

			continue
		}

		results[name] = result
	}

	return results, nil
}
