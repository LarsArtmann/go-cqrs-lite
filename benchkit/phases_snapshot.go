package benchkit

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// maxSnapshotStreams caps the number of streams benchmarked in the snapshot
// phase. The phase creates multiple decider repositories and loads each stream
// several times, so a smaller cap keeps it fast while still producing stable
// latency distributions.
const maxSnapshotStreams = 50

// snapshotPhase measures decider Load performance under three strategies (M16):
//   - Cold replay: plain repository, full event replay every Load.
//   - Snapshot load: snapshot store + EveryNEvents(1), snapshot + delta fold.
//   - Cache hit/miss: state cache, first Load is a miss (full replay), second
//     is a hit (LoadFromVersion of 0 delta events).
//
// Verifies state and version equality across all strategies (correctness).
// Requires event.Store (bundle.EventStore()); gracefully skips otherwise.
// Snapshot load is skipped when the bundle has no SnapshotStore.
func (r *runner) snapshotPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil // ctx done; graceful skip
	}

	store, ok := r.bundle.EventStore()
	if !ok {
		return nil
	}

	d := counterDecider()

	streamCount := min(r.config.Profile.Streams, maxSnapshotStreams)
	if streamCount <= 0 {
		streamCount = 1
	}

	streamIDs := r.aggIDs[:streamCount]

	if err := r.populateSnapshots(ctx, store, d, streamIDs); err != nil {
		return err
	}

	coldStates, coldVersions, err := r.benchmarkColdLoad(ctx, store, d, streamIDs)
	if err != nil {
		return err
	}

	r.benchmarkSnapshotLoad(ctx, store, d, streamIDs, coldStates, coldVersions)

	return r.benchmarkCache(ctx, store, d, streamIDs, coldStates, coldVersions)
}

// populateSnapshots writes one extra event per stream via a snapshot repo
// (EveryNEvents(1)) so a snapshot exists at each stream's latest version. All
// subsequent loads see the same event count.
func (r *runner) populateSnapshots(
	ctx context.Context,
	store event.Store,
	d decider.Decider[CounterState],
	streamIDs []id.StreamID,
) error {
	if r.bundle.SnapshotStore == nil {
		return nil
	}

	strategy, err := snapshot.EveryNEvents(1)
	if err != nil {
		return err
	}

	repo, err := decider.NewRepository[CounterState](store, nil, d,
		decider.WithSnapshotStore[CounterState](r.bundle.SnapshotStore),
		decider.WithCodec[CounterState](r.codec),
		decider.WithSnapshotStrategy[CounterState](strategy),
		decider.WithLoadCoalescing[CounterState](false),
	)
	if err != nil {
		return err
	}

	for _, sid := range streamIDs {
		if ctx.Err() != nil {
			return nil // ctx done; graceful skip
		}

		err := repo.Execute(ctx, sid, benchStreamType,
			func(_ CounterState, ver event.Version) ([]event.Event, error) {
				evt, eErr := event.New(
					benchEventType, sid, benchStreamType,
					ver.Add(1), r.gen.Payload(),
					event.WithCodec(r.codec),
				)
				if eErr != nil {
					return nil, eErr
				}

				return []event.Event{evt}, nil
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// benchmarkColdLoad measures full-replay Load latency with a plain repository.
// Returns the baseline states and versions for correctness comparison.
func (r *runner) benchmarkColdLoad(
	ctx context.Context,
	store event.Store,
	d decider.Decider[CounterState],
	streamIDs []id.StreamID,
) ([]CounterState, []event.Version, error) {
	repo, err := decider.NewRepository[CounterState](store, nil, d,
		decider.WithLoadCoalescing[CounterState](false),
	)
	if err != nil {
		return nil, nil, err
	}

	coll := NewLatencyCollector(0)
	states := make([]CounterState, len(streamIDs))
	versions := make([]event.Version, len(streamIDs))

	for i, sid := range streamIDs {
		if ctx.Err() != nil {
			return nil, nil, nil // ctx done; graceful skip
		}

		start := time.Now()

		state, ver, loadErr := repo.Load(ctx, sid, benchStreamType)
		coll.Record(time.Since(start))

		if loadErr != nil {
			return nil, nil, loadErr
		}

		states[i] = state
		versions[i] = ver
	}

	r.result.SnapshotColdLatency = coll.Stats()

	return states, versions, nil
}

// benchmarkSnapshotLoad measures snapshot-backed Load latency and verifies
// correctness against the cold-load baseline. Skipped when no SnapshotStore.
func (r *runner) benchmarkSnapshotLoad(
	ctx context.Context,
	store event.Store,
	d decider.Decider[CounterState],
	streamIDs []id.StreamID,
	coldStates []CounterState,
	coldVersions []event.Version,
) {
	if r.bundle.SnapshotStore == nil {
		return
	}

	strategy, err := snapshot.EveryNEvents(1)
	if err != nil {
		return
	}

	repo, err := decider.NewRepository[CounterState](store, nil, d,
		decider.WithSnapshotStore[CounterState](r.bundle.SnapshotStore),
		decider.WithCodec[CounterState](r.codec),
		decider.WithSnapshotStrategy[CounterState](strategy),
		decider.WithLoadCoalescing[CounterState](false),
	)
	if err != nil {
		return
	}

	coll := NewLatencyCollector(0)

	for i, sid := range streamIDs {
		if ctx.Err() != nil {
			break // ctx done; report partial
		}

		start := time.Now()

		state, ver, loadErr := repo.Load(ctx, sid, benchStreamType)
		coll.Record(time.Since(start))

		if loadErr != nil {
			return
		}

		if state != coldStates[i] || ver != coldVersions[i] {
			r.result.SnapshotCorrectnessErrors++
		}
	}

	r.result.SnapshotLoadLatency = coll.Stats()
}

// benchmarkCache measures cache-miss (first load, full replay) and cache-hit
// (second load, delta fold of 0 events) latency with correctness checks.
func (r *runner) benchmarkCache(
	ctx context.Context,
	store event.Store,
	d decider.Decider[CounterState],
	streamIDs []id.StreamID,
	coldStates []CounterState,
	coldVersions []event.Version,
) error {
	repo, err := decider.NewRepository[CounterState](store, nil, d,
		decider.WithLoadCoalescing[CounterState](false),
		decider.WithStateCache[CounterState](decider.NewStateCache[CounterState](len(streamIDs)*2)),
	)
	if err != nil {
		return err
	}

	// Cache miss: first load of each stream (full replay, populates cache).
	missColl := NewLatencyCollector(0)

	for i, sid := range streamIDs {
		if ctx.Err() != nil {
			return nil // ctx done; graceful skip
		}

		start := time.Now()

		state, ver, loadErr := repo.Load(ctx, sid, benchStreamType)
		missColl.Record(time.Since(start))

		if loadErr != nil {
			return loadErr
		}

		if state != coldStates[i] || ver != coldVersions[i] {
			r.result.SnapshotCorrectnessErrors++
		}
	}

	r.result.CacheMissLatency = missColl.Stats()

	// Cache hit: second load (LoadFromVersion of 0 delta events).
	hitColl := NewLatencyCollector(0)

	for i, sid := range streamIDs {
		if ctx.Err() != nil {
			break // ctx done; report partial
		}

		start := time.Now()

		state, ver, loadErr := repo.Load(ctx, sid, benchStreamType)
		hitColl.Record(time.Since(start))

		if loadErr != nil {
			return loadErr
		}

		if state != coldStates[i] || ver != coldVersions[i] {
			r.result.SnapshotCorrectnessErrors++
		}
	}

	r.result.CacheHitLatency = hitColl.Stats()

	return nil
}
