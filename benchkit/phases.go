package benchkit

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// rawSinkPhase pre-builds all events and then times only EventSink.Save,
// isolating backend write capacity from event generation and encoding overhead.
// Uses the main bundle but writes to SEPARATE stream IDs so it does not
// conflict with the write phase's streams.
//
// Boundary: the timed region starts at the first Save call and ends after the
// last Save returns. Event creation, payload generation, codec encoding, ID
// generation, and metadata construction are all performed BEFORE timing begins.
// This produces RawSinkLatency and RawSinkThroughput — the pure storage cost.
//
// Note: raw sink events are written to the same store and will appear in the
// journal. They are NOT counted in Result.TotalEvents (which reflects only
// the write phase). Tests that assert journal contents or replay event counts
// should set Config.SkipRawSink = true.
func (r *runner) rawSinkPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	if r.bundle.EventSink == nil {
		return nil
	}

	profile := r.config.Profile

	// Separate stream IDs for raw sink measurement.
	rawIDs := make([]id.StreamID, profile.Streams)
	rawRefs := make([]id.StreamRef, profile.Streams)

	for i := range profile.Streams {
		rawIDs[i] = id.NewStreamID()
		rawRefs[i] = id.NewStreamRef(benchStreamType, rawIDs[i])
	}

	// Pre-build all events (not timed).
	type prebuiltBatch struct {
		ref     id.StreamRef
		events  []event.Event
		version event.Version
	}

	allBatches := make([][]prebuiltBatch, profile.Streams)

	for i := range profile.Streams {
		var version event.Version

		written := 0

		for written < profile.EventsPerStream {
			batchSize := min(profile.BatchSize, profile.EventsPerStream-written)

			events, err := r.createBatch(rawIDs[i], version, batchSize)
			if err != nil {
				return err
			}

			allBatches[i] = append(allBatches[i], prebuiltBatch{
				ref:     rawRefs[i],
				events:  events,
				version: version,
			})

			version = version.Add(uint(batchSize))
			written += batchSize
		}
	}

	// Time only the Save calls.
	coll := NewLatencyCollector(0)

	var totalEvents atomic.Int64

	start := time.Now()

	err := runConcurrent(
		ctx, profile.Streams, r.concurrency,
		func(ctx context.Context, aggIdx int) error {
			for _, batch := range allBatches[aggIdx] {
				startSave := time.Now()

				if err := r.bundle.EventSink.Save(
					ctx,
					batch.ref,
					batch.events,
					batch.version,
				); err != nil {
					return err
				}

				coll.Record(time.Since(startSave))
				totalEvents.Add(int64(len(batch.events)))
			}

			return nil
		},
	)

	elapsed := time.Since(start)
	r.result.RawSinkLatency = coll.Stats()

	if elapsed > 0 && err == nil {
		r.result.RawSinkThroughput = float64(totalEvents.Load()) / elapsed.Seconds()
	}

	// Context cancellation during raw sink is not fatal — the main phases
	// still produced their measurements.
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; not fatal
	}

	return err
}

// writePhase writes events to all streams concurrently and collects
// write latency percentiles plus overall throughput.
func (r *runner) writePhase(ctx context.Context) error {
	coll := NewLatencyCollector(0)

	var totalEvents atomic.Int64

	profile := r.config.Profile
	start := time.Now()

	err := runConcurrent(
		ctx, profile.Streams, r.concurrency,
		func(ctx context.Context, aggIdx int) error {
			return r.writeOneAggregate(ctx, aggIdx, coll, &totalEvents)
		},
	)

	elapsed := time.Since(start)
	r.result.WriteLatency = coll.Stats()
	r.result.TotalEvents = int(totalEvents.Load())

	if elapsed > 0 && err == nil {
		r.result.WriteThroughput = float64(totalEvents.Load()) / elapsed.Seconds()
	}

	return err
}

func (r *runner) writeOneAggregate(
	ctx context.Context,
	aggIdx int,
	coll *LatencyCollector,
	total *atomic.Int64,
) error {
	ref := r.refs[aggIdx]
	aggID := r.aggIDs[aggIdx]
	profile := r.config.Profile

	var version event.Version

	written := 0

	for written < profile.EventsPerStream {
		if ctx.Err() != nil {
			return nil //nolint:nilerr // ctx done; graceful skip
		}

		batchSize := min(profile.BatchSize, profile.EventsPerStream-written)

		events, err := r.createBatch(aggID, version, batchSize)
		if err != nil {
			return err
		}

		start := time.Now()

		if err := r.bundle.EventSink.Save(ctx, ref, events, version); err != nil {
			return err
		}

		coll.Record(time.Since(start))

		version = version.Add(uint(batchSize))
		written += batchSize
		total.Add(int64(batchSize))
	}

	return nil
}

func (r *runner) createBatch(
	aggID id.StreamID,
	version event.Version,
	batchSize int,
) ([]event.Event, error) {
	events := make([]event.Event, batchSize)

	for j := range batchSize {
		evt, err := event.New(
			benchEventType, aggID, benchStreamType,
			version.Add(uint(j+1)), r.gen.Payload(),
			event.WithCodec(r.codec),
		)
		if err != nil {
			return nil, err
		}

		events[j] = evt
	}

	return events, nil
}
