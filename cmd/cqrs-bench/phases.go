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

	fmt.Println("Benchmark phases (execution order):")
	fmt.Println()

	for _, name := range benchkit.PhaseNames() {
		desc := descriptions[name]
		if desc == "" {
			desc = "(no description)"
		}

		fmt.Printf("  %-18s %s\n", name, desc)
	}

	fmt.Println()
	fmt.Println("Phases are skipped when:")
	fmt.Println("  - A config flag disables them (--skip-*, --replay)")
	fmt.Println(
		"  - The bundle lacks a required component (no EventSink, no CheckpointStore, etc.)",
	)
	fmt.Println("  - --strict fails the run if ANY phase is skipped")

	return nil
}
