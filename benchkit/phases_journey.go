package benchkit

import (
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// maxJourneySamples caps the number of end-to-end journey measurements so the
// phase stays bounded even on large profiles. Each sample writes one event,
// synchronously projects it, and dispatches a typed query.
const maxJourneySamples = 200

// journeyPhase measures the end-to-end publish→projection→query journey (M14).
//
// For each sample it:
//  1. Writes a single journey event to a fresh stream (EventSink.Save).
//  2. Synchronously applies the event through a projection (projection.Handle),
//     materializing a counter into the read model (Get + Set per event).
//  3. Dispatches a typed query that reads the materialized counter.
//
// JourneyLatency records the full round trip (Save → project → query result).
// JourneyProjectionLatency isolates the projection.Handle leg. JourneyQueryLatency
// isolates the query-dispatch leg.
//
// The projectionhost is a batch-drainer (catches up and exits), so the journey
// uses synchronous projection.Handle to measure per-event materialization cost
// without poll-interval artifacts. The existing projectionPhase already
// benchmarks the batch-drain path through the host.
//
// Requires EventSink + ReadModels. Gracefully skips when either is absent.
func (r *runner) journeyPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil // ctx done; graceful skip
	}

	if r.bundle.EventSink == nil || r.bundle.ReadModels == nil {
		return nil
	}

	store := r.bundle.ReadModels

	// The projection writes per-stream counters via Get + Set (real I/O).
	proj := newJourneyProjection(store)

	// Pre-allocate the journey stream IDs so the query dispatcher knows them.
	sampleCount := min(r.config.Profile.Streams, maxJourneySamples)
	if sampleCount <= 0 {
		sampleCount = 1
	}

	streamIDs := make([]id.StreamID, sampleCount)
	for i := range sampleCount {
		streamIDs[i] = id.NewStreamID()
	}

	disp := newBenchQueryDispatcher(store, streamIDs)
	defer disp.Close()

	journeyColl := NewLatencyCollector(0)
	projColl := NewLatencyCollector(0)
	queryColl := NewLatencyCollector(0)

	var correctnessErrors int

	for i := range sampleCount {
		if ctx.Err() != nil {
			break // ctx done; report partial results
		}

		streamID := streamIDs[i]
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

		// 1. Write the event to the store.
		if err := r.bundle.EventSink.Save(
			ctx, ref, []event.Event{evt}, event.Version(0),
		); err != nil {
			return err
		}

		// 2. Synchronously project the event (materialize counter).
		projStart := time.Now()

		if err := proj.Handle(ctx, evt); err != nil {
			return errorfamily.WrapTransient(err, "benchkit.journey_project",
				"journey projection handle")
		}

		projColl.Record(time.Since(projStart))

		// 3. Query the materialized value.
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
	r.result.JourneyProjectionLatency = projColl.Stats()
	r.result.JourneyQueryLatency = queryColl.Stats()
	r.result.JourneySamples = int(r.result.JourneyLatency.Count)
	r.result.QueryCorrectnessErrors += correctnessErrors

	return nil
}
