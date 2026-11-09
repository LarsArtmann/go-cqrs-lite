package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

func listPhasesHandler(_ context.Context, _ *AppConfig, _ *ListPhasesFlags) error {
	descriptions := map[string]string{
		"write":          "Concurrent event writes (Save with optimistic concurrency)",
		"batch-write":    "Batch event writes (AppendBatch, no concurrency checks)",
		"read":           "Event load latency (cold + warm passes)",
		"versioned-read": "Point-in-time recovery reads (LoadFromVersion, LoadToVersion, LoadToTimestamp)",
		"read-model":     "KV store Set/Get latency for read-model projections",
		"projection":     "Projection host replay throughput and lag",
		"checkpoint":     "Checkpoint Save/Load latency (projection recovery)",
		"mixed-workload": "Concurrent read-during-write contention",
		"journey":        "End-to-end publish→projection→query round-trip",
		"query":          "Typed query dispatch latency (hit, miss, paginated)",
		"snapshot":       "Snapshot/cache hit-rate and cold-replay comparison",
		"metaengine":     "Metaengine planner overhead (counter + map ADTs)",
	}

	// phaseMetrics names the metrics each phase feeds. Benchstat names (the
	// lines --format benchstat emits, universe: benchkit.MetricNames) are
	// listed directly; metrics that only appear in the text/JSON report are
	// prefixed "report:". A phase whose metrics are all zero emits nothing.
	phaseMetrics := map[string]string{
		"write": fmt.Sprintf(
			"%s %s %s %s write_tail_ratio rawsink_throughput rawsink_p50_ns rawsink_p99_ns",
			benchkit.MetricWriteThroughput,
			benchkit.MetricWriteP50NS,
			benchkit.MetricWriteP99NS,
			benchkit.MetricWriteMaxNS,
		),
		"batch-write":    "report: batchWriteLatency, batchWriteThroughput",
		"read":           "load_p50_ns load_p99_ns load_max_ns cold_read_p50_ns cold_read_p99_ns tail_ratio",
		"versioned-read": "report: loadFromVersionLatency, loadToVersionLatency, loadToTimestampLatency",
		"read-model":     "report: readModelSet, readModelGet",
		"projection":     "report: projectionEvents, projectionLag",
		"checkpoint":     "report: checkpointSaveLatency, checkpointLoadLatency",
		"mixed-workload": "report: mixedWorkload.writeLatency, mixedWorkload.readLatency",
		"journey":        "journey_p50_ns journey_p99_ns journey_projection_p99_ns journey_query_p99_ns",
		"query":          "query_hit_p50_ns query_hit_p99_ns query_miss_p99_ns query_paginated_p99_ns",
		"snapshot":       "snapshot_cold_p50_ns snapshot_cold_p99_ns snapshot_load_p99_ns cache_miss_p99_ns cache_hit_p99_ns",
		"metaengine":     "metaengine_scan_p99_ns metaengine_point_read_p99_ns metaengine_apply_concurrent",
	}

	fmt.Println("Benchmark phases (execution order):")
	fmt.Println()

	for _, name := range benchkit.PhaseNames() {
		desc := descriptions[name]
		if desc == "" {
			desc = "(no description)"
		}

		fmt.Printf("  %-18s %s\n", name, desc)
		fmt.Printf("  %-18s metrics: %s\n", "", phaseMetrics[name])
	}

	fmt.Println()
	fmt.Println("Benchstat names appear in --format benchstat output; report: metrics")
	fmt.Println("appear in the text/JSON report only. Zero-valued metrics (skipped or")
	fmt.Println("unsupported phase) are always omitted.")
	fmt.Println()
	fmt.Println("Phases are skipped when:")
	fmt.Println("  - A config flag disables them (--skip-*, --replay)")
	fmt.Println(
		"  - The bundle lacks a required component (no EventSink, no CheckpointStore, etc.)",
	)
	fmt.Println("  - --strict fails the run if ANY phase is skipped")

	return nil
}
