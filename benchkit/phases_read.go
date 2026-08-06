package benchkit

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// readPhase loads all streams concurrently, then runs journal scans.
// The number of read passes scales with Profile.ReadRatio so that read-heavy
// profiles perform more reads than write-heavy ones.
//
// The first pass is recorded separately as ColdReadLatency — it captures
// disk I/O latency when the OS page cache is cold. Subsequent passes hit
// the page cache and represent warm latency. LoadLatency aggregates ALL
// passes (cold + warm).
func (r *runner) readPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	coll := NewLatencyCollector(0)
	profile := r.config.Profile

	readPasses := readPassesFor(profile.ReadRatio)

	for pass := 0; pass < readPasses && ctx.Err() == nil; pass++ {
		passColl := NewLatencyCollector(0)

		err := runConcurrent(
			ctx, profile.Streams, r.concurrency,
			func(ctx context.Context, aggIdx int) error {
				ref := r.refs[aggIdx]
				start := time.Now()

				_, err := r.bundle.EventSource.Load(ctx, ref)
				if err != nil {
					return err
				}

				elapsed := time.Since(start)
				coll.Record(elapsed)
				passColl.Record(elapsed)

				return nil
			},
		)

		// Capture first pass as cold-read latency.
		if pass == 0 {
			r.result.ColdReadLatency = passColl.Stats()
		}

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
		r.recordSkip("read model phase", "bundle has no ReadModels (kv.Store)")

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
