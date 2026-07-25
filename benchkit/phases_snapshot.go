package benchkit

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
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
		return nil //nolint:nilerr // ctx done; graceful skip
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

	// ── Populate snapshots ──
	// Execute one command per stream via a snapshot repo (EveryNEvents(1)) so a
	// snapshot exists at each stream's latest version. This writes one extra
	// event per stream — all subsequent loads see the same event count.
	if r.bundle.SnapshotStore != nil {
		strategy, err := snapshot.EveryNEvents(1)
		if err != nil {
			return err
		}

		snapRepo, err := decider.NewRepository[CounterState](store, nil, d,
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
				return nil //nolint:nilerr // ctx done; graceful skip
			}

			err := snapRepo.Execute(ctx, sid, benchStreamType,
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
	}

	// ── Cold Load: plain repo, full replay ──
	coldRepo, err := decider.NewRepository[CounterState](store, nil, d,
		decider.WithLoadCoalescing[CounterState](false),
	)
	if err != nil {
		return err
	}

	coldColl := NewLatencyCollector(0)
	coldStates := make([]CounterState, streamCount)
	coldVersions := make([]event.Version, streamCount)

	for i, sid := range streamIDs {
		if ctx.Err() != nil {
			return nil //nolint:nilerr // ctx done; graceful skip
		}

		start := time.Now()

		state, ver, loadErr := coldRepo.Load(ctx, sid, benchStreamType)
		coldColl.Record(time.Since(start))

		if loadErr != nil {
			return loadErr
		}

		coldStates[i] = state
		coldVersions[i] = ver
	}

	r.result.SnapshotColdLatency = coldColl.Stats()

	// ── Snapshot Load: snapshot + delta fold ──
	if r.bundle.SnapshotStore != nil {
		strategy, err := snapshot.EveryNEvents(1)
		if err != nil {
			return err
		}

		snapLoadRepo, err := decider.NewRepository[CounterState](store, nil, d,
			decider.WithSnapshotStore[CounterState](r.bundle.SnapshotStore),
			decider.WithCodec[CounterState](r.codec),
			decider.WithSnapshotStrategy[CounterState](strategy),
			decider.WithLoadCoalescing[CounterState](false),
		)
		if err != nil {
			return err
		}

		snapColl := NewLatencyCollector(0)

		for i, sid := range streamIDs {
			if ctx.Err() != nil {
				break //nolint:nilerr // ctx done; report partial
			}

			start := time.Now()

			state, ver, loadErr := snapLoadRepo.Load(ctx, sid, benchStreamType)
			snapColl.Record(time.Since(start))

			if loadErr != nil {
				return loadErr
			}

			// Correctness: snapshot load must match cold load exactly.
			if state != coldStates[i] || ver != coldVersions[i] {
				r.result.SnapshotCorrectnessErrors++
			}
		}

		r.result.SnapshotLoadLatency = snapColl.Stats()
	}

	// ── Cache miss + hit ──
	cacheRepo, err := decider.NewRepository[CounterState](store, nil, d,
		decider.WithLoadCoalescing[CounterState](false),
		decider.WithStateCache[CounterState](decider.NewStateCache[CounterState](streamCount*2)),
	)
	if err != nil {
		return err
	}

	// Cache miss: first load of each stream (full replay, populates cache).
	missColl := NewLatencyCollector(0)

	for i, sid := range streamIDs {
		if ctx.Err() != nil {
			return nil //nolint:nilerr // ctx done; graceful skip
		}

		start := time.Now()

		state, ver, loadErr := cacheRepo.Load(ctx, sid, benchStreamType)
		missColl.Record(time.Since(start))

		if loadErr != nil {
			return loadErr
		}

		// Correctness: cache miss must match cold load.
		if state != coldStates[i] || ver != coldVersions[i] {
			r.result.SnapshotCorrectnessErrors++
		}
	}

	r.result.CacheMissLatency = missColl.Stats()

	// Cache hit: second load of each stream (LoadFromVersion of 0 delta).
	hitColl := NewLatencyCollector(0)

	for i, sid := range streamIDs {
		if ctx.Err() != nil {
			break //nolint:nilerr // ctx done; report partial
		}

		start := time.Now()

		state, ver, loadErr := cacheRepo.Load(ctx, sid, benchStreamType)
		hitColl.Record(time.Since(start))

		if loadErr != nil {
			return loadErr
		}

		// Correctness: cache hit must match cold load.
		if state != coldStates[i] || ver != coldVersions[i] {
			r.result.SnapshotCorrectnessErrors++
		}
	}

	r.result.CacheHitLatency = hitColl.Stats()

	return nil
}
