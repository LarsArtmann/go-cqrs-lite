package benchkit

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// checkpointPhase benchmarks Save/Load latency on the CheckpointStore.
// Projection hosts checkpoint after every batch of events — a slow
// CheckpointStore directly increases projection lag during replay.
func (r *runner) checkpointPhase(ctx context.Context) error {
	cpStore := r.bundle.CheckpointStore
	if r.skipPhase(ctx, "checkpoint phase", "bundle has no CheckpointStore", cpStore != nil) {
		return nil
	}

	sampleCount := min(len(r.refs), maxCheckpointSamples)
	saveColl := NewLatencyCollector(0)
	loadColl := NewLatencyCollector(0)

	for range sampleCount {
		if err := ctx.Err(); err != nil {
			return err
		}

		projName := "bench-checkpoint"
		checkpoint := event.Checkpoint{
			EventID:     id.NewEventID(),
			ProcessedAt: time.Now(),
		}

		start := time.Now()
		err := cpStore.Save(ctx, projName, checkpoint)

		saveColl.Record(time.Since(start))

		if err != nil {
			return err
		}

		start = time.Now()
		_, err = cpStore.Load(ctx, projName)

		loadColl.Record(time.Since(start))

		if err != nil {
			return err
		}
	}

	r.result.CheckpointSaveLatency = saveColl.Stats()
	r.result.CheckpointLoadLatency = loadColl.Stats()

	return nil
}

const maxCheckpointSamples = 50
