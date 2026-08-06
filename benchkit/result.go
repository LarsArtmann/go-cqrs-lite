package benchkit

import "time"

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
	RawSinkLatency    LatencyStats `json:"rawSinkLatency"`
	RawSinkThroughput float64      `json:"rawSinkThroughput,omitempty"`

	// Write metrics — generated events timed including generation + encoding + Save.
	WriteLatency    LatencyStats `json:"writeLatency"`
	WriteThroughput float64      `json:"writeThroughput"`

	// Read metrics
	// LoadLatency aggregates ALL read passes. ColdReadLatency isolates
	// the first pass (OS page cache miss → disk I/O). On SQLite/Pebble,
	// ColdReadLatency P50 may be 10x higher than the warm LoadLatency P50.
	LoadLatency     LatencyStats  `json:"loadLatency"`
	ColdReadLatency LatencyStats  `json:"coldReadLatency"`
	ReadAllTime     time.Duration `json:"readAllTime"`
	ReadFromTime    time.Duration `json:"readFromTime"`

	// Versioned read metrics (point-in-time recovery performance)
	// LoadFromVersionLatency measures loading events after a specific version.
	// LoadToVersionLatency measures loading events up to a specific version.
	// LoadToTimestampLatency measures loading events up to a timestamp.
	LoadFromVersionLatency LatencyStats `json:"loadFromVersionLatency"`
	LoadToVersionLatency   LatencyStats `json:"loadToVersionLatency"`
	LoadToTimestampLatency LatencyStats `json:"loadToTimestampLatency"`

	// Read model metrics (raw kv.Store Set/Get)
	ReadModelGet LatencyStats `json:"readModelGet"`
	ReadModelSet LatencyStats `json:"readModelSet"`

	// Projection metrics (zero-valued when no projections ran)
	ProjectionLag    time.Duration `json:"projectionLag"`
	ProjectionEvents int64         `json:"projectionEvents"`

	// Journey metrics — end-to-end publish→projection→query latency (M14).
	// JourneyLatency times the full round trip per event: Save → projection.Handle
	// (materialize) → typed query returns the updated value. The three component
	// latencies are also reported individually.
	// Zero-valued when Config.SkipJourney is true or the bundle lacks
	// EventSink + ReadModels.
	JourneyLatency           LatencyStats `json:"journeyLatency"`
	JourneyProjectionLatency LatencyStats `json:"journeyProjectionLatency"`
	JourneyQueryLatency      LatencyStats `json:"journeyQueryLatency"`
	JourneySamples           int          `json:"journeySamples,omitempty"`

	// Query dispatch metrics — typed query.Dispatcher overhead (M15).
	// QueryHitLatency: registered handler found and invoked.
	// QueryMissLatency: unregistered type (handler-not-found error path).
	// QueryPaginatedLatency: paginated result construction.
	// Zero-valued when Config.SkipQuery is true or ReadModels is absent.
	QueryHitLatency        LatencyStats `json:"queryHitLatency"`
	QueryMissLatency       LatencyStats `json:"queryMissLatency"`
	QueryPaginatedLatency  LatencyStats `json:"queryPaginatedLatency"`
	QueryCorrectnessErrors int          `json:"queryCorrectnessErrors,omitempty"`

	// Snapshot/cache metrics — decider Load performance under strategies (M16).
	// SnapshotColdLatency: full replay, no snapshot, no cache.
	// SnapshotLoadLatency: snapshot load + delta fold (zero when no SnapshotStore).
	// CacheMissLatency: first Load (populates cache, full replay).
	// CacheHitLatency: second Load (cache hit, delta fold of 0 new events).
	// SnapshotCorrectnessErrors: count of state/version mismatches across strategies.
	SnapshotColdLatency       LatencyStats `json:"snapshotColdLatency"`
	SnapshotLoadLatency       LatencyStats `json:"snapshotLoadLatency"`
	CacheMissLatency          LatencyStats `json:"cacheMissLatency"`
	CacheHitLatency           LatencyStats `json:"cacheHitLatency"`
	SnapshotCorrectnessErrors int          `json:"snapshotCorrectnessErrors,omitempty"`

	// Metaengine metrics — planner overhead with counter + map workloads (M17).
	// MetaEngineApplyLatency: per-event Apply latency through the cost-based
	//   planner (fold dispatch + engine write). Counter ADT.
	// MetaEngineQueryLatency: ExecuteTyped read latency (engine point read +
	//   result materialization). Counter ADT.
	// MetaEngineApplyThroughput: events/sec sustained during the Apply burst.
	// MetaEngineScanLatency: TypedReader.Scan latency with filter — the primary
	//   collection read path. Shows O(N) scan cost at the configured scale.
	// MetaEnginePointReadLatency: TypedReader.Get latency — single-item point
	//   lookup through the planner. Different code path from ExecuteTyped.
	// MetaEngineApplyConcurrent: concurrent Apply throughput (events/sec with
	//   Config.Concurrency goroutines). Lower than single-threaded throughput
	//   indicates lock contention in the engine.
	// MetaEngineScanResults: items returned by the scan (correctness check).
	// Zero-valued when Config.SkipMetaEngine is true or the bundle has no metaengine.
	MetaEngineApplyLatency     LatencyStats `json:"metaEngineApplyLatency"`
	MetaEngineQueryLatency     LatencyStats `json:"metaEngineQueryLatency"`
	MetaEngineApplyThroughput  float64      `json:"metaEngineApplyThroughput,omitempty"`
	MetaEngineScanLatency      LatencyStats `json:"metaEngineScanLatency"`
	MetaEnginePointReadLatency LatencyStats `json:"metaEnginePointReadLatency"`
	MetaEngineApplyConcurrent  float64      `json:"metaEngineApplyConcurrent,omitempty"`
	MetaEngineScanResults      int          `json:"metaEngineScanResults,omitempty"`

	// Recovery metrics (zero-valued when Config.Recovery is false).
	// RecoveryTime measures the wall-clock time to close the store,
	// reopen it via the factory, and load all streams — simulating
	// crash-recovery replay. RecoveredEvents is the total events loaded.
	RecoveryTime    time.Duration `json:"recoveryTime,omitempty"`
	RecoveredEvents int           `json:"recoveredEvents,omitempty"`

	// Mixed workload metrics — concurrent reads + writes against the same
	// store. WriteLatency is measured under read contention; ReadLatency is
	// measured under write contention. Zero-valued when Config.SkipMixed is
	// true or the bundle lacks EventSink + EventSource.
	MixedWorkload MixedResult `json:"mixedWorkload,omitempty"` //nolint:modernize // omitzero needs go1.27

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

	// RepeatMean is the arithmetic mean of throughput across repeat runs.
	RepeatMean float64 `json:"repeatMean,omitempty"`

	// RepeatStdDev is the population standard deviation of throughput samples.
	// Measures how spread out the runs are. Lower is better.
	RepeatStdDev float64 `json:"repeatStdDev,omitempty"`

	// RepeatCoV is the coefficient of variation (StdDev / Mean) as a fraction.
	// The key reliability indicator:
	//   CoV < 0.05  — results are trustworthy (<5% noise)
	//   CoV 0.05–0.15 — usable but verify individual metrics
	//   CoV > 0.15  — too noisy for reliable comparison; increase Repeat
	RepeatCoV float64 `json:"repeatCoV,omitempty"`

	// RepeatIsReliable is true when CoV < 0.10 (10%). When false, do NOT
	// trust the median for cross-backend comparison — increase Repeat.
	RepeatIsReliable bool `json:"repeatIsReliable,omitempty"`

	// GC pause metrics — garbage collection behavior during the benchmark.
	// GC pauses are the dominant cause of P99 latency spikes. These metrics
	// reveal whether tail latency is caused by the backend or by Go's GC.
	// GCCount is the number of GC cycles during the benchmark.
	// GCTotalPause is the cumulative stop-the-world pause time.
	// GCMaxPause is the longest single GC pause observed.
	// GCMeanPause is the average pause per GC cycle.
	GCCount      int           `json:"gcCount"`
	GCTotalPause time.Duration `json:"gcTotalPause"`
	GCMaxPause   time.Duration `json:"gcMaxPause"`
	GCMeanPause  time.Duration `json:"gcMeanPause"`

	// Allocation metrics — total heap allocations during the benchmark.
	// High alloc rates correlate with GC pressure and latency variance.
	// AllocCount is the cumulative number of heap allocations.
	// AllocBytes is the cumulative bytes allocated.
	AllocCount uint64 `json:"allocCount"`
	AllocBytes uint64 `json:"allocBytes"`

	// IntegrityErrors is the count of events that failed read-back
	// verification after the write phase. Zero means all written events
	// were read back correctly. Non-zero indicates data corruption.
	IntegrityErrors int `json:"integrityErrors,omitempty"`

	// Derived metrics — computed from raw metrics for easy comparison.
	// These are the decision-grade rates that eliminate manual division.

	// AllocsPerOp is the average heap allocations per event written.
	// Directly predicts GC pressure: higher allocs → more frequent GC.
	AllocsPerOp float64 `json:"allocsPerOp,omitempty"`

	// BytesPerOp is the average bytes allocated per event written.
	// Measures per-event allocation footprint (payload + encoding overhead).
	BytesPerOp float64 `json:"bytesPerOp,omitempty"`

	// GCPercent is the percentage of wall-clock time spent in GC pauses.
	// GCPercent > 5 means GC is a significant performance factor.
	GCPercent float64 `json:"gcPercent,omitempty"`

	// TailRatio is LoadLatency.P99 / LoadLatency.P50. A ratio > 3 means
	// tail latency is unpredictable — the P50 looks good but real users
	// experience 3x+ worse at the 99th percentile.
	TailRatio float64 `json:"tailRatio,omitempty"`

	// WriteTailRatio is WriteLatency.P99 / WriteLatency.P50. Same concept as
	// TailRatio but for the write path. High write tail ratios matter for
	// ingestion-sensitive workloads where a single slow write stalls the
	// pipeline.
	WriteTailRatio float64 `json:"writeTailRatio,omitempty"`

	// Metaengine SQLite comparison — the same Map ADT workload run against
	// the SQLite engine (NewSQLiteEngine). This gives a direct Memory vs
	// SQLite comparison: Memory shows the planner+fold overhead with zero
	// I/O, SQLite shows the cost of SQL query execution + JSON extraction.
	// Zero-valued when Config.SkipMetaEngine is true.
	MetaEngineSQLiteApplyThroughput  float64      `json:"metaEngineSQLiteApplyThroughput,omitempty"`
	MetaEngineSQLiteScanLatency      LatencyStats `json:"metaEngineSQLiteScanLatency"`
	MetaEngineSQLitePointReadLatency LatencyStats `json:"metaEngineSQLitePointReadLatency"`

	// SkippedPhases lists phase names that were skipped, either because a Config
	// skip flag was set (SkipReads, SkipMetaEngine, etc.) or because the bundle
	// lacked a required component (e.g. no SnapshotStore). Every skip is also
	// recorded as a Warning with a human-readable reason.
	SkippedPhases []string `json:"skippedPhases,omitempty"`

	// Warnings holds advisory messages about phases that were skipped, partially
	// run, or otherwise degraded. Never empty when SkippedPhases is non-empty.
	Warnings []string `json:"warnings,omitempty"`

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

	// WriteAmplification is the ratio of on-disk bytes to logical event
	// bytes (DatabaseBytes / EventBytes). A ratio of 1.0 means zero
	// storage overhead; 3.0 means the backend writes 3x the logical data.
	// This is THE key metric for comparing storage efficiency across
	// backends — an LSM-tree may write 10x while a row-store writes 2x.
	WriteAmplification float64 `json:"writeAmplification"`
}

// MixedResult holds concurrent read + write metrics from the mixed workload
// phase. The key insight: WriteLatency under read contention and ReadLatency
// under write contention reveal whether the backend can serve both paths
// simultaneously without degradation.
type MixedResult struct {
	WriteLatency LatencyStats `json:"writeLatency"`
	ReadLatency  LatencyStats `json:"readLatency"`
	WriteOps     int64        `json:"writeOps"`
	ReadOps      int64        `json:"readOps"`
	WriteErrors  int64        `json:"writeErrors,omitempty"`
	ReadErrors   int64        `json:"readErrors,omitempty"`
	Writers      int          `json:"writers"`
	Readers      int          `json:"readers"`
}
