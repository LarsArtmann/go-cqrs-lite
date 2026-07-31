package benchkit

import (
	"context"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// mixedWorkloadPhase runs writers and readers concurrently against the same
// store. Writers append events to fresh streams; readers load events from the
// streams written by the preceding [writePhase]. This is the only phase that
// measures read latency UNDER write contention — the real production question
// ("can the backend serve reads while writes hammer it?").
//
// Writer concurrency = Config.Concurrency.
// Reader concurrency = max(1, int(Concurrency * ReadRatio)).
// Writers run to completion (Streams * EventsPerStream / BatchSize operations);
// readers run continuously until writers finish, then stop.
func (r *runner) mixedWorkloadPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	if r.bundle.EventSink == nil || r.bundle.EventSource == nil {
		return nil
	}

	// Need existing streams to read from (written by writePhase).
	if len(r.refs) == 0 {
		return nil
	}

	profile := r.config.Profile

	// Fresh streams for the mixed-phase writers so we don't need version
	// tracking for existing streams.
	mixedStreams := min(profile.Streams,
		// cap to keep phase duration reasonable
		1000)

	mixedRefs := make([]id.StreamRef, mixedStreams)
	mixedStreamIDs := make([]id.StreamID, mixedStreams)

	for i := range mixedRefs {
		sid := id.NewStreamID()
		mixedStreamIDs[i] = sid
		mixedRefs[i] = id.NewStreamRef(benchStreamType, sid)
	}

	writeColl := NewLatencyCollector(mixedStreams)
	readColl := NewLatencyCollector(0) // unbounded — readers run until cancelled

	var (
		writeOps    atomic.Int64
		readOps     atomic.Int64
		readErrors  atomic.Int64
		writeErrors atomic.Int64
	)

	// Reader goroutines: continuously load random existing streams until
	// the writer context is done.
	readerCtx, readerCancel := context.WithCancel(ctx)
	defer readerCancel()

	readerCount := max(int(float64(r.concurrency)*profile.ReadRatio), 1)

	var readerWg sync.WaitGroup

	for i := range readerCount {
		readerWg.Go(func() {
			// Each reader goroutine needs its own *rand.Rand — math/rand/v2
			// (*Rand) is NOT safe for concurrent use.
			localRng := rand.New(rand.NewPCG(uint64(r.config.Seed), uint64(i)+1))

			for readerCtx.Err() == nil {
				ref := r.refs[localRng.IntN(len(r.refs))]

				start := time.Now()

				_, err := r.bundle.EventSource.Load(readerCtx, ref)

				readColl.Record(time.Since(start))

				readOps.Add(1)

				if err != nil {
					readErrors.Add(1)
				}
			}
		})
	}

	// Writer goroutines: write to fresh streams using runConcurrent.
	writeErr := runConcurrent(
		ctx, mixedStreams, r.concurrency,
		func(ctx context.Context, idx int) error {
			ref := mixedRefs[idx]

			var version event.Version

			remaining := max(profile.EventsPerStream/profile.BatchSize, 1)

			for range remaining {
				if ctx.Err() != nil {
					return nil
				}

				events, err := r.createBatch(mixedStreamIDs[idx], version, profile.BatchSize)
				if err != nil {
					return err
				}

				start := time.Now()

				if err := r.bundle.EventSink.Save(ctx, ref, events, version); err != nil {
					writeErrors.Add(1)

					continue // skip version advance on error
				}

				writeColl.Record(time.Since(start))
				writeOps.Add(int64(len(events)))
				version = version.Add(uint(len(events)))
			}

			return nil
		},
	)

	// Stop readers and wait for them to finish.
	readerCancel()
	readerWg.Wait()

	if writeErr != nil {
		return errorfamily.WrapTransient(
			writeErr,
			"benchkit.mixed_workload",
			"mixed workload write phase",
		)
	}

	r.result.MixedWorkload = MixedResult{
		WriteLatency: writeColl.Stats(),
		ReadLatency:  readColl.Stats(),
		WriteOps:     writeOps.Load(),
		ReadOps:      readOps.Load(),
		WriteErrors:  writeErrors.Load(),
		ReadErrors:   readErrors.Load(),
		Writers:      r.concurrency,
		Readers:      readerCount,
	}

	return nil
}
