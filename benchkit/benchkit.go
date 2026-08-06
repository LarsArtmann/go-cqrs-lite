package benchkit

import (
	"encoding/json/v2"
	"fmt"
	"io"
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

	// CPUModel is the CPU model string (e.g. "AMD Ryzen 9 7950X").
	// Empty when unavailable. Different CPU models produce dramatically
	// different latency numbers — this field makes cross-machine
	// comparisons honest.
	CPUModel string `json:"cpuModel,omitempty"`

	// TotalRAMBytes is the total system RAM in bytes. Empty/zero when
	// unavailable. RAM affects page-cache behavior and thus read latency.
	TotalRAMBytes uint64 `json:"totalRamBytes,omitempty"`
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
	//
	// Excluded from JSON serialization (interfaces cannot round-trip);
	// [Config.CodecName] provides the serializable representation and resolves
	// back to a concrete Codec during JSON unmarshal.
	Codec codec.Codec `json:"-"`

	// CodecName is the JSON-serializable encoding name derived from Codec
	// (e.g. "json", "cbor"). Empty when Codec is nil; resolved back to a
	// concrete Codec during JSON unmarshal via [codec.ForEncoding].
	CodecName string

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

	// SkipJourney skips the end-to-end publish→projection→query journey
	// phase (M14). The journey phase writes single events to fresh streams,
	// synchronously projects them into the read model, and dispatches typed
	// queries — measuring full round-trip latency. Skipped automatically when
	// the bundle lacks EventSink + ReadModels.
	SkipJourney bool

	// SkipQuery skips the typed query dispatch phase (M15). The query phase
	// benchmarks query.Dispatcher overhead (hit, miss, paginated paths) against
	// a pre-populated read model. Skipped automatically when ReadModels is absent.
	SkipQuery bool

	// SkipSnapshot skips the snapshot/cache hit-rate phase (M16). The snapshot
	// phase measures decider Load performance under cold replay, snapshot load,
	// and cache-hit strategies with correctness assertions. Skipped automatically
	// when the bundle's EventSink does not implement event.Store.
	SkipSnapshot bool

	// SkipMetaEngine skips the metaengine phase (M17). The metaengine phase
	// benchmarks the cost-based storage planner's Apply throughput and
	// ExecuteTyped read latency with a counter workload. Skipped automatically
	// when the bundle has no metaengine (MetaEngine() returns nil).
	SkipMetaEngine bool

	// SkipRawSink skips the raw prebuilt-event sink phase that isolates
	// EventSink.Save throughput from event generation/encoding overhead.
	// When false (default), the runner pre-builds all events, then times
	// only the Save calls — producing RawSinkLatency and RawSinkThroughput
	// that are independent of generator and codec cost.
	SkipRawSink bool

	// SkipMixed skips the mixed read-during-write phase. When false (default),
	// the runner runs N writer goroutines and M reader goroutines concurrently
	// against the same store, measuring read latency under write contention
	// and write latency under read contention. Reader count is derived from
	// Profile.ReadRatio * Config.Concurrency.
	SkipMixed bool

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

	// ProgressWriter, when non-nil, receives debounced progress updates
	// during benchmark execution. Phase transitions are always reported;
	// in-phase heartbeat updates fire every ProgressInterval. Use this for
	// long-running benchmarks (stress, large profiles) to get real-time
	// visibility into which phase is running and how long it has taken.
	// Typically os.Stderr. Zero ProgressInterval defaults to 5s.
	ProgressWriter   io.Writer      `json:"-"`
	ProgressInterval time.Duration
}

// validate checks that the Config has required fields set.
func (c *Config) validate() error {
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

// MarshalJSON serializes Config with Codec replaced by CodecName, enabling
// JSON round-trip. The [codec.Codec] interface cannot round-trip through JSON
// (there is no concrete type information on the wire), so Codec is excluded
// (json:"-") and CodecName carries the encoding string.
func (c Config) MarshalJSON() ([]byte, error) {
	type alias Config

	c.CodecName = codecEncodingName(c.Codec)

	return json.Marshal(alias(c), json.WithMarshalers(durationMarshalers))
}

// UnmarshalJSON deserializes Config, resolving CodecName back to a concrete
// [codec.Codec] via [codec.ForEncoding].
func (c *Config) UnmarshalJSON(data []byte) error {
	type alias Config

	var aux alias

	if err := json.Unmarshal(data, &aux, json.WithUnmarshalers(durationUnmarshalers)); err != nil {
		return fmt.Errorf("unmarshal Config: %w", err)
	}

	*c = Config(aux)

	if c.CodecName != "" {
		resolved, err := codec.ForEncoding(codec.Encoding(c.CodecName))
		if err != nil {
			return fmt.Errorf("Config.CodecName %q: %w", c.CodecName, err)
		}

		c.Codec = resolved
	}

	return nil
}

func codecEncodingName(c codec.Codec) string {
	if c == nil {
		return ""
	}

	return string(c.Encoding())
}
