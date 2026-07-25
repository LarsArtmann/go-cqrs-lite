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
