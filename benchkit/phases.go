package benchkit

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
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

// readPhase loads all streams concurrently, then runs journal scans.
// The number of read passes scales with Profile.ReadRatio so that read-heavy
// profiles perform more reads than write-heavy ones.
func (r *runner) readPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	coll := NewLatencyCollector(0)
	profile := r.config.Profile

	readPasses := readPassesFor(profile.ReadRatio)

	for pass := 0; pass < readPasses && ctx.Err() == nil; pass++ {
		err := runConcurrent(
			ctx, profile.Streams, r.concurrency,
			func(ctx context.Context, aggIdx int) error {
				ref := r.refs[aggIdx]
				start := time.Now()

				_, err := r.bundle.EventSource.Load(ctx, ref)
				if err != nil {
					return err
				}

				coll.Record(time.Since(start))

				return nil
			},
		)
		if err != nil {
			r.result.LoadLatency = coll.Stats()

			return err
		}
	}

	r.result.LoadLatency = coll.Stats()

	r.runJournalScans(ctx)

	return nil
}

// readPassesFor converts a ReadRatio (0.0–1.0) into the number of read passes.
// 0.0–0.1 = 1 pass, 0.5 = 5 passes, 0.8 = 8 passes, 1.0 = 10 passes.
func readPassesFor(ratio float64) int {
	if ratio <= 0 {
		return 1
	}

	passes := max(int(ratio*10+0.5), 1)

	return passes
}

func (r *runner) runJournalScans(ctx context.Context) {
	scans := max(r.config.Profile.JournalScans, 1)

	if r.bundle.Journal != nil {
		start := time.Now()

		for range scans {
			_, _ = r.bundle.Journal.ReadAll(ctx)
		}

		r.result.ReadAllTime = time.Since(start)
	}

	if r.bundle.SeekableJournal != nil {
		var afterID id.EventID

		start := time.Now()

		for range scans {
			_, _ = r.bundle.SeekableJournal.ReadFrom(ctx, afterID, 1000)
		}

		r.result.ReadFromTime = time.Since(start)
	}
}

// readModelPhase benchmarks raw kv.Store Set and Get operations.
func (r *runner) readModelPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	if r.bundle.ReadModels == nil {
		return nil
	}

	profile := r.config.Profile
	store := r.bundle.ReadModels
	setColl := NewLatencyCollector(0)

	payload, err := r.codec.Encode(r.gen.Payload())
	if err != nil {
		return fmt.Errorf("encode benchmark payload: %w", err)
	}

	keys := make([][]byte, profile.Streams)

	for i := range profile.Streams {
		keys[i] = []byte(r.aggIDs[i].String())
	}

	err = runConcurrent(
		ctx, profile.Streams, r.concurrency,
		func(ctx context.Context, idx int) error {
			start := time.Now()

			if err := store.Set(ctx, keys[idx], payload); err != nil {
				return err
			}

			setColl.Record(time.Since(start))

			return nil
		},
	)

	r.result.ReadModelSet = setColl.Stats()

	if err != nil {
		return err
	}

	// If the context was cancelled (e.g. Duration timeout), some keys may
	// not have been Set. Skip the Get phase to avoid spurious kv.ErrNotFound.
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; skip Get phase
	}

	getColl := NewLatencyCollector(0)
	err = runConcurrent(
		ctx, profile.Streams, r.concurrency,
		func(ctx context.Context, idx int) error {
			start := time.Now()

			_, err := store.Get(ctx, keys[idx])
			if err != nil {
				return err
			}

			getColl.Record(time.Since(start))

			return nil
		},
	)

	r.result.ReadModelGet = getColl.Stats()

	return err
}

// projectionPhase runs a counting projection through projectionhost and
// measures lag and throughput. Skipped when SeekableJournal or
// CheckpointStore is absent.
func (r *runner) projectionPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	if r.bundle.SeekableJournal == nil || r.bundle.CheckpointStore == nil {
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
			_ = host.Stop()

			r.collectProjectionStats(host)

			return nil // timeout — report what we got
		case <-ticker.C:
		case <-ctx.Done():
			_ = host.Stop()

			r.collectProjectionStats(host)

			return nil
		}
	}

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

// durabilityPhase measures on-disk storage footprint.
// Tries the DiskSizer interface first (precise per-backend sizing via
// stack.Bundle.DiskSize); falls back to filesystem walk of Config.DiskPath.
// DiskSize() returns -1 when no disk-size reporter is registered (memory,
// SQLite without WithDiskSize), signaling the fallback path.
func (r *runner) durabilityPhase() {
	if sizer, ok := any(r.bundle).(DiskSizer); ok {
		if size := sizer.DiskSize(); size >= 0 {
			r.result.Disk.DatabaseBytes = size

			return
		}
	}

	if r.config.DiskPath != "" {
		r.result.Disk.DatabaseBytes = measureDirSize(r.config.DiskPath)
	}
}

func measureDirSize(path string) int64 {
	var total int64

	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries
		}

		if !info.IsDir() {
			total += info.Size()
		}

		return nil
	})

	return total
}

// recoveryPhase simulates crash recovery: closes the current bundle
// (flushing all writes to disk), reopens it via the factory (reopening at
// the same path for persistent backends), and loads all streams to measure
// replay time. For memory backends, the reopened store is empty, so
// RecoveredEvents will be zero — this is expected and documents that
// memory backends have no crash recovery.
func (r *runner) recoveryPhase(_ context.Context) error {
	// Recovery is a post-benchmark durability check — it must run even if
	// the benchmark context has expired. Use a fresh context so that Load
	// calls are not canceled by the parent's deadline.
	ctx := context.Background() //nolint:contextcheck // intentional fresh context for post-benchmark recovery

	// Close the current bundle to flush all writes.
	_ = r.bundle.Close()
	r.bundle = nil // prevent double-close in teardown

	// Reopen via factory — for persistent backends (SQLite, Pebble),
	// the factory reopens at the same path and all events are recovered.
	recovered, err := r.factory()
	if err != nil {
		return fmt.Errorf("recovery factory: %w", err)
	}

	if recovered == nil || recovered.EventSource == nil {
		if recovered != nil {
			_ = recovered.Close()
		}

		return nil
	}

	defer func() { _ = recovered.Close() }()

	start := time.Now()
	totalEvents := 0

	for _, ref := range r.refs {
		events, err := recovered.EventSource.Load(ctx, ref)
		if err != nil {
			// Memory backend: streams don't exist after reopen — skip.
			continue
		}

		totalEvents += len(events)
	}

	r.result.RecoveryTime = time.Since(start)
	r.result.RecoveredEvents = totalEvents

	return nil
}
