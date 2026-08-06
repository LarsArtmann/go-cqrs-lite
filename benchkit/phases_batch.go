package benchkit

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// batchWritePhase benchmarks AppendBatch throughput for multi-event writes.
// Batch writes are a fundamentally different performance profile from single
// Save calls — backends may batch-fsync, amortize transaction overhead, or
// use copy-on-write optimizations that only apply to multi-event transactions.
func (r *runner) batchWritePhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	if r.bundle.EventSink == nil {
		r.recordSkip("batch write phase", "bundle has no EventSink")
		return nil
	}

	sink := r.bundle.EventSink
	profile := r.config.Profile

	batchSize := min(profile.EventsPerStream, maxBatchSize)
	if batchSize < 2 {
		r.warn("batch write phase: skipped (profile has < 2 events per stream)")
		return nil
	}

	sampleCount := min(len(r.refs), maxBatchSamples)
	coll := NewLatencyCollector(0)
	totalEvents := 0
	startAll := time.Now()

	for i := range sampleCount {
		if ctx.Err() != nil {
			break //nolint:nilerr // ctx done; return partial results
		}

		ref := id.NewStreamRef(fmt.Sprintf("BatchEntity-%d", i),
			id.NewStreamID())

		events := make([]event.Event, 0, batchSize)
		for j := range batchSize {
			evt, err := event.NewEvent(
				fmt.Sprintf("batch.event.%d", j),
				ref.ID, ref.Type, event.Version(j+1),
				generatePayload(profile.PayloadBytes),
			)
			if err != nil {
				return err
			}

			events = append(events, evt)
		}

		start := time.Now()
		err := sink.AppendBatch(ctx, ref, events)
		coll.Record(time.Since(start))
		if err != nil {
			return err
		}

		totalEvents += len(events)
	}

	elapsed := time.Since(startAll).Seconds()
	if elapsed > 0 && totalEvents > 0 {
		r.result.BatchWriteThroughput = float64(totalEvents) / elapsed
	}

	r.result.BatchWriteLatency = coll.Stats()

	return nil
}

const (
	maxBatchSize   = 50
	maxBatchSamples = 20
)
