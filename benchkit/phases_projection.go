package benchkit

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// projectionPhase runs a counting projection through projectionhost and
// measures lag and throughput. Skipped when SeekableJournal or
// CheckpointStore is absent.
func (r *runner) projectionPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil // duration expired; partial results are valid
	}

	if r.bundle.SeekableJournal == nil || r.bundle.CheckpointStore == nil {
		r.recordSkip("projection phase",
			"bundle missing SeekableJournal or CheckpointStore")

		return nil
	}

	proj := newCountingProjection(r.bundle.ReadModels)

	host, err := projectionhost.New(
		r.bundle.SeekableJournal,
		r.bundle.CheckpointStore,
		projectionhost.WithBatchSize(500),
	)
	if err != nil {
		return err
	}

	if err := host.Register(proj); err != nil {
		return err
	}

	if err := host.Start(ctx); err != nil {
		return err
	}

	// Poll until the projection catches up to all written events, then stop.
	// This gives the worker time to actually process events instead of being
	// immediately cancelled.
	target := int64(r.result.TotalEvents)

	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		var processed int64
		for _, ws := range host.Status() {
			processed += ws.Processed
		}

		if processed >= target {
			break
		}

		select {
		case <-deadline.C:
			//cqrs-lint:ignore(C023) library code or intentional pattern
			_ = host.Stop()

			r.collectProjectionStats(host)

			return nil // timeout — report what we got
		case <-ticker.C:
		case <-ctx.Done():
			//cqrs-lint:ignore(C023) library code or intentional pattern
			_ = host.Stop()

			r.collectProjectionStats(host)

			return nil
		}
	}

	//cqrs-lint:ignore(C023) library code or intentional pattern
	_ = host.Stop()

	r.collectProjectionStats(host)

	return nil
}

// newCountingProjection creates a projection that increments a per-stream
// counter for each event. When a kv.Store is available (bundle.ReadModels),
// the counter is persisted via Get+Set — measuring real projection I/O cost.
// When no kv.Store is available, an in-memory atomic counter is used as a
// fallback (still exercises the projection host machinery, but without I/O).
func newCountingProjection(store kv.Store) projection.Projection {
	if store != nil {
		return newKVCountingProjection(store)
	}

	var count atomic.Int64

	return projection.NewProjection(
		"bench-counter",
		func(_ context.Context, _ event.Event) error {
			count.Add(1)

			return nil
		},
		[]event.Type{benchEventType},
	)
}

// newKVCountingProjection creates a projection that persists a per-stream
// event count to a kv.Store. Each event triggers a Get (current count) +
// Set (incremented count), measuring real projection write amplification.
func newKVCountingProjection(store kv.Store) projection.Projection {
	return projection.NewProjection(
		"bench-counter",
		func(ctx context.Context, evt event.Event) error {
			key := []byte("bench:count:" + evt.StreamID().String())

			val, err := store.Get(ctx, key)
			if err != nil && !errors.Is(err, kv.ErrNotFound) {
				return fmt.Errorf("projection Get: %w", err)
			}

			var n uint64
			if len(val) > 0 {
				n = binary.BigEndian.Uint64(val)
			}

			n++

			var buf [8]byte
			binary.BigEndian.PutUint64(buf[:], n)

			if err := store.Set(ctx, key, buf[:]); err != nil {
				return fmt.Errorf("projection Set: %w", err)
			}

			return nil
		},
		[]event.Type{benchEventType},
	)
}

func (r *runner) collectProjectionStats(host *projectionhost.Host) {
	r.result.ProjectionLag = host.LagDuration()

	for _, ws := range host.Status() {
		r.result.ProjectionEvents += ws.Processed
	}
}
