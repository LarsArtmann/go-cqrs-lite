package main

import (
	"context"
	"fmt"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// phaseDoc documents one benchmark phase for the list-phases command.
type phaseDoc struct {
	// name is the phase's identifier.
	name string
	// label is how reports name the phase in skippedPhases and warnings.
	label string
	// summary says what the phase measures.
	summary string
	// metrics are the benchstat names the phase feeds; "report:" marks
	// metrics that only appear in the text and JSON report.
	metrics string
	// flags are the command-line flags that skip the phase.
	flags []string
	// skip turns the phase's skip flag on in a benchkit.Config. A phase
	// that only --replay skips has none.
	skip func(*benchkit.Config)
}

// phaseDocs lists every benchmark phase in benchkit's execution order.
// TestPhaseDocs pins the order against benchkit.PhaseNames and every flag
// against a real run, so the listing cannot drift from what the runner does.
//
// The raw-sink phase is listed although benchkit.PhaseNames omits it: the
// runner executes it and --skip-raw-sink exists.
func phaseDocs() []phaseDoc {
	skipReads := func(config *benchkit.Config) { config.SkipReads = true }
	skipProjections := func(config *benchkit.Config) { config.SkipProjections = true }

	return []phaseDoc{
		{
			name: "write", label: "write phase",
			summary: "Concurrent event writes (Save with optimistic concurrency)",
			metrics: fmt.Sprintf("%s %s %s %s write_tail_ratio",
				benchkit.MetricWriteThroughput, benchkit.MetricWriteP50NS,
				benchkit.MetricWriteP99NS, benchkit.MetricWriteMaxNS),
			flags: []string{"--replay"},
		},
		{
			name: "batch-write", label: "batch write phase",
			summary: "Batch event writes (AppendBatch, no concurrency checks)",
			metrics: "report: batchWriteLatency, batchWriteThroughput",
			flags:   []string{"--replay", "--skip-batch-write"},
			skip:    func(config *benchkit.Config) { config.SkipBatchWrite = true },
		},
		{
			name: "read", label: "read phase",
			summary: "Event load latency (cold and warm passes)",
			metrics: "load_p50_ns load_p99_ns load_max_ns cold_read_p50_ns cold_read_p99_ns tail_ratio",
			flags:   []string{"--skip-reads"},
			skip:    skipReads,
		},
		{
			name: "versioned-read", label: "versioned read phase",
			summary: "Point-in-time recovery reads (LoadFromVersion, LoadToVersion, LoadToTimestamp)",
			metrics: "report: loadFromVersionLatency, loadToVersionLatency, loadToTimestampLatency",
			flags:   []string{"--skip-reads"},
			skip:    skipReads,
		},
		{
			name: "read-model", label: "read model phase",
			summary: "KV store Set and Get latency for read-model projections",
			metrics: "report: readModelSet, readModelGet",
			flags:   []string{"--skip-read-models"},
			skip:    func(config *benchkit.Config) { config.SkipReadModels = true },
		},
		{
			name: "projection", label: "projection phase",
			summary: "Projection host replay throughput and lag",
			metrics: "report: projectionEvents, projectionLag",
			flags:   []string{"--skip-projections"},
			skip:    skipProjections,
		},
		{
			name: "checkpoint", label: "checkpoint phase",
			summary: "Checkpoint Save and Load latency (projection recovery)",
			metrics: "report: checkpointSaveLatency, checkpointLoadLatency",
			flags:   []string{"--skip-projections"},
			skip:    skipProjections,
		},
		{
			name: "mixed-workload", label: "mixed workload phase",
			summary: "Concurrent read-during-write contention",
			metrics: "report: mixedWorkload.writeLatency, mixedWorkload.readLatency",
			flags:   []string{"--replay", "--skip-mixed"},
			skip:    func(config *benchkit.Config) { config.SkipMixed = true },
		},
		{
			name: "journey", label: "journey phase",
			summary: "End-to-end publish, projection, and query round-trip",
			metrics: "journey_p50_ns journey_p99_ns journey_projection_p99_ns journey_query_p99_ns",
			flags:   []string{"--replay", "--skip-journey"},
			skip:    func(config *benchkit.Config) { config.SkipJourney = true },
		},
		{
			name: "query", label: "query phase",
			summary: "Typed query dispatch latency (hit, miss, paginated)",
			metrics: "query_hit_p50_ns query_hit_p99_ns query_miss_p99_ns query_paginated_p99_ns",
			flags:   []string{"--skip-query"},
			skip:    func(config *benchkit.Config) { config.SkipQuery = true },
		},
		{
			name: "snapshot", label: "snapshot phase",
			summary: "Snapshot and cache hit-rate phase with cold-replay comparison",
			metrics: "snapshot_cold_p50_ns snapshot_cold_p99_ns snapshot_load_p99_ns cache_miss_p99_ns cache_hit_p99_ns",
			flags:   []string{"--replay", "--skip-snapshot"},
			skip:    func(config *benchkit.Config) { config.SkipSnapshot = true },
		},
		{
			name: "metaengine", label: "metaengine phase",
			summary: "Metaengine planner overhead (counter and map ADTs)",
			metrics: "metaengine_scan_p99_ns metaengine_point_read_p99_ns metaengine_apply_concurrent",
			flags:   []string{"--skip-metaengine"},
			skip:    func(config *benchkit.Config) { config.SkipMetaEngine = true },
		},
		{
			name: "raw-sink", label: "raw sink phase",
			summary: "Raw prebuilt-event sink (isolates EventSink.Save throughput from generation and encoding cost)",
			metrics: "rawsink_throughput rawsink_p50_ns rawsink_p99_ns",
			flags:   []string{"--replay", "--skip-raw-sink"},
			skip:    func(config *benchkit.Config) { config.SkipRawSink = true },
		},
	}
}

// listPhasesHandler is the list-phases command: the phase catalog with the
// flags that skip each phase.
func listPhasesHandler(context.Context, *appConfig, cmdguard.NoFlags) error {
	fmt.Println("Benchmark phases (execution order):")
	fmt.Println()

	for _, phase := range phaseDocs() {
		fmt.Printf("  %-18s %s\n", phase.name, phase.summary)
		fmt.Printf("  %-18s metrics: %s\n", "", phase.metrics)
		fmt.Printf("  %-18s skipped by: %s\n", "", strings.Join(phase.flags, ", "))
	}

	fmt.Println()
	fmt.Println("Benchstat names appear in --format benchstat output; report: metrics appear")
	fmt.Println("in the text and JSON report only. Zero-valued metrics (a skipped or")
	fmt.Println("unsupported phase) are always omitted.")
	fmt.Println()
	fmt.Println("A phase is also skipped when the bundle lacks a required component (no")
	fmt.Println("EventSink, no CheckpointStore, ...); reports name it by its label, such as")
	fmt.Println("\"read model phase\". --strict fails the run when a phase the backend can")
	fmt.Println("run is skipped, and with --repeat 2 or more when a headline metric is NOISY.")

	return nil
}
