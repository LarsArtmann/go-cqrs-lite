package benchkit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// writePhase writes events to all aggregates concurrently and collects
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

// readPhase loads all aggregates concurrently, then runs journal scans.
// The number of read passes scales with Profile.ReadRatio so that read-heavy
// profiles perform more reads than write-heavy ones.
func (r *runner) readPhase(ctx context.Context) error {
	coll := NewLatencyCollector(0)
	profile := r.config.Profile

	readPasses := readPassesFor(profile.ReadRatio)

	for range readPasses {
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
	if r.bundle.Journal != nil {
		start := time.Now()
		_, _ = r.bundle.Journal.ReadAll(ctx)
		r.result.ReadAllTime = time.Since(start)
	}

	if r.bundle.SeekableJournal != nil {
		var afterID id.EventID

		start := time.Now()
		_, _ = r.bundle.SeekableJournal.ReadFrom(ctx, afterID, 1000)
		r.result.ReadFromTime = time.Since(start)
	}
}

// readModelPhase benchmarks raw kv.Store Set and Get operations.
func (r *runner) readModelPhase(ctx context.Context) error {
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
	if r.bundle.SeekableJournal == nil || r.bundle.CheckpointStore == nil {
		return nil
	}

	proj := projection.NewProjection(
		"bench-counter",
		func(_ context.Context, _ event.Event) error { return nil },
		[]event.Type{benchEventType},
	)

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

	_ = host.Stop()

	r.collectProjectionStats(host)

	return nil
}

func (r *runner) collectProjectionStats(host *projectionhost.Host) {
	r.result.ProjectionLag = host.LagDuration()

	for _, ws := range host.Status() {
		r.result.ProjectionEvents += ws.Processed
	}
}

// durabilityPhase measures on-disk storage footprint.
// Tries the DiskSizer interface first (precise per-backend sizing),
// falls back to filesystem walk of Config.DiskPath.
func (r *runner) durabilityPhase() {
	if sizer, ok := any(r.bundle).(DiskSizer); ok {
		r.result.Disk.DatabaseBytes = sizer.DiskSize()
		return
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
