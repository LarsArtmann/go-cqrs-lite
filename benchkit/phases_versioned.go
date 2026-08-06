package benchkit

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// versionedReadPhase benchmarks point-in-time recovery reads:
// LoadFromVersion, LoadToVersion, and LoadToTimestamp.
//
// These methods are the core value proposition of event sourcing — the ability
// to reconstruct state at any point in time. This phase measures their latency
// independently from the full-stream Load measured in the read phase.
//
// Requires EventSource. Gracefully skips when absent or when no streams exist.
func (r *runner) versionedReadPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	if r.bundle.EventSource == nil || len(r.refs) == 0 {
		r.recordSkip("versioned read phase",
			"bundle has no EventSource or no streams written")

		return nil
	}

	source := r.bundle.EventSource
	profile := r.config.Profile

	sampleCount := min(len(r.refs), maxVersionedReadSamples)
	halfVersion := event.Version(profile.EventsPerStream / 2)
	now := time.Now()

	fromColl := NewLatencyCollector(0)
	toColl := NewLatencyCollector(0)
	tsColl := NewLatencyCollector(0)

	for i := range sampleCount {
		if err := ctx.Err(); err != nil {
			return err
		}

		ref := r.refs[i]

		start := time.Now()
		_, err := source.LoadFromVersion(ctx, ref, halfVersion)

		fromColl.Record(time.Since(start))

		if err != nil {
			return err
		}

		start = time.Now()
		_, err = source.LoadToVersion(ctx, ref, halfVersion)

		toColl.Record(time.Since(start))

		if err != nil {
			return err
		}

		start = time.Now()
		_, err = source.LoadToTimestamp(ctx, ref, now)

		tsColl.Record(time.Since(start))

		if err != nil {
			return err
		}
	}

	r.result.LoadFromVersionLatency = fromColl.Stats()
	r.result.LoadToVersionLatency = toColl.Stats()
	r.result.LoadToTimestampLatency = tsColl.Stats()

	return nil
}

const maxVersionedReadSamples = 50
