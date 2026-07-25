package benchkit

import (
	"context"
	"errors"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// maxJourneySamples caps the number of end-to-end journey measurements so the
// phase stays bounded even on large profiles. Each sample writes one event,
// polls for projection materialization, and dispatches a typed query.
const maxJourneySamples = 200

// journeyCatchupTimeout is the maximum time to wait for the projection host to
// process a single event during warm-up or a journey sample. Generous because
// the host polls the journal on a short interval.
const journeyCatchupTimeout = 10 * time.Second

// journeyPhase measures the end-to-end publish→projection→query journey (M14).
//
// For each sample it:
//  1. Writes a single journey event to a fresh stream (EventSink.Save).
//  2. Polls the read model until the projection materializes the event.
//  3. Dispatches a typed query that reads the materialized counter.
//
// JourneyLatency records the full round trip (Save → queryable → query result).
// JourneyQueryLatency isolates the query-dispatch leg (projection already
// caught up).
//
// Requires SeekableJournal + CheckpointStore + ReadModels. Gracefully skips
// when any are absent.
func (r *runner) journeyPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	if r.bundle.SeekableJournal == nil ||
		r.bundle.CheckpointStore == nil ||
		r.bundle.ReadModels == nil {
		return nil
	}

	proj := newJourneyProjection(r.bundle.ReadModels)

	host, err := projectionhost.New(
		r.bundle.SeekableJournal,
		r.bundle.CheckpointStore,
		projectionhost.WithBatchSize(100),
	)
	if err != nil {
		return err
	}

	if err := host.Register(proj); err != nil {
		_ = host.Stop()

		return err
	}

	if err := host.Start(ctx); err != nil {
		_ = host.Stop()

		return err
	}

	defer host.Stop()

	// Warm-up: write one throwaway journey event and wait for the projection
	// to materialize it. This confirms the host has scanned past the
	// write-phase journal and is in live-tail mode before timed samples begin.
	if err := r.waitForJourneyCatchup(ctx, host); err != nil {
		return errorfamily.WrapTransient(err, "benchkit.journey_warmup",
			"journey projection warm-up")
	}

	// Set up the query dispatcher against the read model.
	sampleCount := min(r.config.Profile.Streams, maxJourneySamples)
	if sampleCount <= 0 {
		sampleCount = 1
	}

	streamIDs := make([]id.StreamID, sampleCount)
	disp := newBenchQueryDispatcher(r.bundle.ReadModels, streamIDs)
	defer disp.Close()

	journeyColl := NewLatencyCollector(0)
	queryColl := NewLatencyCollector(0)
	var correctnessErrors int

	for i := range sampleCount {
		if ctx.Err() != nil {
			break //nolint:nilerr // ctx done; report partial results
		}

		streamID := id.NewStreamID()
		streamIDs[i] = streamID
		ref := id.NewStreamRef(benchStreamType, streamID)

		evt, err := event.New(
			journeyEventType, streamID, benchStreamType,
			event.Version(1), r.gen.Payload(),
			event.WithCodec(r.codec),
		)
		if err != nil {
			return err
		}

		start := time.Now()

		if err := r.bundle.EventSink.Save(
			ctx, ref, []event.Event{evt}, event.Version(0),
		); err != nil {
			return err
		}

		// Poll until the projection materializes this stream's counter.
		if !r.pollJourneyMaterialized(ctx, streamID) {
			// Timeout — projection did not catch up. Record the journey latency
			// up to the timeout but skip the query leg.
			journeyColl.Record(time.Since(start))

			correctnessErrors++

			continue
		}

		// Query the materialized value (isolated timing).
		queryStart := time.Now()

		result, qErr := query.DispatchTyped[CountResult](
			ctx, disp, getCountQuery{streamID: streamID.String()},
		)

		queryColl.Record(time.Since(queryStart))
		journeyColl.Record(time.Since(start))

		if qErr != nil {
			correctnessErrors++

			continue
		}

		// Each journey stream has exactly one event → count must be 1.
		if result.Count != 1 {
			correctnessErrors++
		}
	}

	r.result.JourneyLatency = journeyColl.Stats()
	r.result.JourneyQueryLatency = queryColl.Stats()
	r.result.JourneySamples = int(r.result.JourneyLatency.Count)
	r.result.QueryCorrectnessErrors += correctnessErrors

	return nil
}

// waitForJourneyCatchup writes a throwaway journey event and polls until the
// projection materializes it, confirming the host is live and caught up.
func (r *runner) waitForJourneyCatchup(
	ctx context.Context,
	host *projectionhost.Host,
) error {
	warmupID := id.NewStreamID()
	ref := id.NewStreamRef(benchStreamType, warmupID)

	evt, err := event.New(
		journeyEventType, warmupID, benchStreamType,
		event.Version(1), r.gen.Payload(),
		event.WithCodec(r.codec),
	)
	if err != nil {
		return err
	}

	if err := r.bundle.EventSink.Save(
		ctx, ref, []event.Event{evt}, event.Version(0),
	); err != nil {
		return err
	}

	if !r.pollJourneyMaterialized(ctx, warmupID) {
		return errors.New("projection did not materialize warm-up event")
	}

	return nil
}

// pollJourneyMaterialized polls the read model until the given stream's counter
// appears (projection caught up) or the catch-up timeout expires.
func (r *runner) pollJourneyMaterialized(
	ctx context.Context,
	streamID id.StreamID,
) bool {
	key := countKey(streamID.String())
	deadline := time.NewTimer(journeyCatchupTimeout)
	defer deadline.Stop()

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		_, ok, err := readCount(ctx, r.bundle.ReadModels, key)
		if err != nil {
			return false
		}

		if ok {
			return true
		}

		select {
		case <-deadline.C:
			return false
		case <-ticker.C:
		case <-ctx.Done():
			return false
		}
	}
}
