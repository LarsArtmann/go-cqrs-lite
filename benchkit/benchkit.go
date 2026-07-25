package benchkit

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// Factory creates a fresh [stack.Bundle] for benchmarking. Each call should
// produce an independent Bundle with its own backing resources (temp directory,
// etc.). The caller is responsible for calling Bundle.Close after the run.
type Factory func() (*stack.Bundle, error)

// DiskSizer is implemented by *stack.Bundle to report precise on-disk size.
// The runner checks DiskSize() first; a return of -1 means "not available"
// (e.g. memory backend, SQLite without WithDiskSize), and the runner falls
// back to walking Config.DiskPath. Disk-backed presets (Pebble) register
// a disk-size function via stack.WithDiskSize at construction time.
type DiskSizer interface {
	DiskSize() int64
}

// SchemaVersion is the version of the Result JSON schema. Increment when
// the Result struct's JSON shape changes in a backward-incompatible way.
const SchemaVersion = "1.0.0"

// Environment captures machine and runtime metadata for reproducibility.
// Every result records the exact context it was produced in so that
// comparisons across machines, Go versions, or CPU limits are honest.
type Environment struct {
	GoVersion  string `json:"goVersion"`
	NumCPU     int    `json:"numCPU"`
	GOMAXPROCS int    `json:"gomaxprocs"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
}

// Config defines a benchmark run.
type Config struct {
	// Profile controls the scale: number of streams, events per stream,
	// concurrency level, read/write ratio, and batch size.
	// Use a named [Profile] or customize the fields directly.
	Profile Profile

	// PayloadSize controls the synthetic payload byte size per event.
	// Default: 256 bytes.
	PayloadSize int

	// PayloadSizes, when non-empty, overrides PayloadSize and produces a MIXED
	// distribution: each event payload is sized by a uniform random pick from
	// this slice. Models real workloads where events vary from small status
	// updates to large events with embedded collections. Default (nil) uses
	// PayloadSize for every event. The distribution mean is reported in
	// Result.PayloadBytes and the full distribution in Result.PayloadSizes.
	PayloadSizes []int

	// Codec controls payload encoding for event creation.
	// nil defaults to [codec.JSONCodec].
	Codec codec.Codec

	// Duration caps the wall-clock time. Zero means run to completion.
	Duration time.Duration

	// Warmup runs this many write+load cycles before measurement begins
	// to warm caches and JIT compilation. Zero disables warmup.
	//
	// Warmup uses a SEPARATE Bundle (the factory is called a second time) so
	// warmup events never pollute the measurement store's journal or metrics.
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

	// SkipReads skips the read phase (stream loads + journal scans).
	SkipReads bool

	// SkipReadModels skips the read-model Set/Get benchmark.
	SkipReadModels bool

	// SkipProjections skips the projection phase.
	SkipProjections bool

	// SkipRawSink skips the raw prebuilt-event sink phase that isolates
	// EventSink.Save throughput from event generation/encoding overhead.
	// When false (default), the runner pre-builds all events, then times
	// only the Save calls — producing RawSinkLatency and RawSinkThroughput
	// that are independent of generator and codec cost.
	SkipRawSink bool

	// Recovery enables the durability recovery phase: after all other
	// phases complete, the runner closes the bundle, reopens it via the
	// factory (reopening at the same path), and loads all streams to
	// measure crash-recovery replay time. Only meaningful for persistent
	// backends (SQLite, Pebble); memory backends produce zero recovery
	// events. Result.RecoveryTime and Result.RecoveredEvents are populated.
	Recovery bool

	// ReplayOnly skips the write phase and benchmarks read/projection
	// performance against an existing store with real data. The runner
	// discovers streams from the Journal (ReadAll) or SeekableJournal
	// (ReadFrom), then loads each stream and runs journal scans +
	// projections. Profile.Streams caps the number of streams loaded.
	// The factory must produce a Bundle that already contains data.
	// Requires Journal or SeekableJournal — returns ErrIncompleteBundle
	// if neither is available.
	ReplayOnly bool

	// Repeat runs the benchmark N times and reports the median result with
	// min/max throughput spread. Zero or 1 means single run (default).
	// Useful because single-run throughput has ~20-25% variance on the
	// memory backend. When Repeat > 1, Result.RepeatCount/Min/Max/Samples
	// are populated on the median result.
	Repeat int
}

// validate checks that the Config has required fields set.
func (c Config) validate() error {
	if c.Profile.Streams <= 0 {
		return fmt.Errorf(
			"%w: Profile.Streams must be > 0, got %d",
			ErrInvalidConfig,
			c.Profile.Streams,
		)
	}

	if c.Profile.EventsPerStream <= 0 {
		return fmt.Errorf("%w: Profile.EventsPerStream must be > 0, got %d",
			ErrInvalidConfig, c.Profile.EventsPerStream)
	}

	if c.Profile.BatchSize <= 0 {
		return fmt.Errorf("%w: Profile.BatchSize must be > 0, got %d",
			ErrInvalidConfig, c.Profile.BatchSize)
	}

	if c.Warmup < 0 {
		return fmt.Errorf("%w: Warmup must be >= 0, got %d", ErrInvalidConfig, c.Warmup)
	}

	return nil
}

// Result is the output of a single benchmark run against one backend.
type Result struct {
	// Identification
	Backend       string        `json:"backend"`
	Profile       string        `json:"profile"`
	Timestamp     time.Time     `json:"timestamp"`
	Duration      time.Duration `json:"duration"`
	SchemaVersion string        `json:"schemaVersion"`
	Environment   Environment   `json:"environment"`

	// Workers is the actual concurrency used (Profile.Concurrency overridden
	// by Config.Concurrency when non-zero). Reported separately from
	// GOMAXPROCS so consumers can distinguish goroutine count from CPU limit.
	Workers int `json:"workers"`

	// Workload
	Streams         int `json:"aggregates"`
	EventsPerStream int `json:"eventsPerAggregate"`
	TotalEvents     int `json:"totalEvents"`
	PayloadBytes    int `json:"payloadBytesPerEvent"`

	// PayloadSizes is the per-event payload-size distribution. Empty for a
	// uniform-size run; populated when Config.PayloadSizes is used (mixed
	// workloads). PayloadBytes holds the mean of this distribution.
	PayloadSizes []int `json:"payloadSizes,omitempty"`

	// WarmupEvents is the number of events written during the warmup phase
	// (on a separate Bundle). Zero when Warmup is disabled.
	WarmupEvents int `json:"warmupEvents,omitempty"`

	// Raw sink metrics — prebuilt events timed against EventSink.Save only.
	// Isolates backend write capacity from event generation and encoding
	// overhead. Zero-valued when Config.SkipRawSink is true.
	RawSinkLatency    LatencyStats `json:"rawSinkLatency"` //nolint:modernize // struct omitempty
	RawSinkThroughput float64      `json:"rawSinkThroughput,omitempty"`

	// Write metrics — generated events timed including generation + encoding + Save.
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

	// Recovery metrics (zero-valued when Config.Recovery is false).
	// RecoveryTime measures the wall-clock time to close the store,
	// reopen it via the factory, and load all streams — simulating
	// crash-recovery replay. RecoveredEvents is the total events loaded.
	RecoveryTime    time.Duration `json:"recoveryTime,omitempty"`
	RecoveredEvents int           `json:"recoveredEvents,omitempty"`

	// Resource metrics
	Memory ResourceStats `json:"memory"`
	CPU    ResourceStats `json:"cpu"`
	Disk   DiskStats     `json:"disk"`

	// Codec used for event payloads
	Codec string `json:"codec"`

	// Repeat info (populated only when Config.Repeat > 1).
	// The Result itself holds the median run's full metrics.
	RepeatCount   int       `json:"repeatCount,omitempty"`
	RepeatMin     float64   `json:"repeatMin,omitempty"`
	RepeatMax     float64   `json:"repeatMax,omitempty"`
	RepeatSamples []float64 `json:"repeatSamples,omitempty"`

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
// is closed automatically after the run. When Warmup > 0, the factory is called
// a second time for a throwaway warmup Bundle that never pollutes measurement.
//
// When Config.Repeat > 1, the benchmark runs N times. Each repeat calls
// factory() fresh, so for in-memory backends each run is fully isolated.
// For persistent backends (SQLite file, Pebble directory), the factory opens
// the same path — meaning later repeats inherit earlier runs' data. To ensure
// isolation with persistent backends, provide a factory that creates a unique
// path per call (e.g., using a temp dir with a unique suffix).
// The returned Result holds the median run's full metrics, annotated
// with min/max throughput across all N runs.
func Run(ctx context.Context, config Config, factory Factory) (*Result, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	if config.Repeat > 1 {
		return runRepeated(ctx, config, factory)
	}

	return newRunner(config, factory).run(ctx)
}

// runRepeated executes the benchmark N times, returning the median result
// annotated with min/max throughput spread.
func runRepeated(ctx context.Context, config Config, factory Factory) (*Result, error) {
	single := config
	single.Repeat = 0

	results := make([]*Result, 0, config.Repeat)

	for i := range config.Repeat {
		r, err := newRunner(single, factory).run(ctx)
		if err != nil {
			return nil, fmt.Errorf("repeat run %d/%d: %w", i+1, config.Repeat, err)
		}

		results = append(results, r)
	}

	// Sort results by throughput so the median index actually corresponds to
	// the median throughput, not insertion order. The previous code sorted a
	// separate samples slice but picked from the unsorted results array.
	sort.Slice(results, func(i, j int) bool {
		return results[i].WriteThroughput < results[j].WriteThroughput
	})

	medianIdx := len(results) / 2
	median := results[medianIdx]

	samples := make([]float64, len(results))
	for i, r := range results {
		samples[i] = r.WriteThroughput
	}

	median.RepeatCount = config.Repeat
	median.RepeatMin = samples[0]
	median.RepeatMax = samples[len(samples)-1]
	median.RepeatSamples = samples

	return median, nil
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
