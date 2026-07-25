package benchkit

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// discoverStreams reads the journal to find existing streams and populates
// r.aggIDs and r.refs. It caps the number of discovered streams at maxStreams.
// Sets r.result.TotalEvents to the total events found in the journal.
//
// When SeekableJournal is available, streams are discovered via batched reads
// (1000 events per batch) to avoid loading the entire journal into memory.
// When only Journal is available, ReadAll is used (loads all events — OOM risk
// for very large stores).
func (r *runner) discoverStreams(ctx context.Context, maxStreams int) error {
	seen := make(map[string]struct{})
	r.aggIDs = make([]id.StreamID, 0, maxStreams)
	r.refs = make([]id.StreamRef, 0, maxStreams)

	var totalEvents int

	switch {
	case r.bundle.SeekableJournal != nil:
		totalEvents = r.discoverFromSeekable(ctx, maxStreams, seen)
	case r.bundle.Journal != nil:
		count, err := r.discoverFromJournal(ctx, maxStreams, seen)
		if err != nil {
			return err
		}

		totalEvents = count
	default:
		return ErrNilBundle
	}

	r.result.TotalEvents = totalEvents
	r.result.Streams = len(r.aggIDs)

	return nil
}

// discoverFromSeekable pages through the SeekableJournal in batches of 1000,
// extracting unique stream IDs. Returns the total event count across all
// batches.
func (r *runner) discoverFromSeekable(
	ctx context.Context,
	maxStreams int,
	seen map[string]struct{},
) int {
	const discoveryBatchSize = 1000

	var afterID id.EventID

	totalEvents := 0

	for ctx.Err() == nil {
		batch, err := r.bundle.SeekableJournal.ReadFrom(ctx, afterID, discoveryBatchSize)
		if err != nil {
			return totalEvents
		}

		if len(batch) == 0 {
			break
		}

		totalEvents += r.collectStreamIDs(batch, maxStreams, seen)
		afterID = batch[len(batch)-1].ID()

		if len(batch) < discoveryBatchSize {
			break // last page
		}
	}

	return totalEvents
}

// discoverFromJournal reads all events via Journal.ReadAll and extracts unique
// stream IDs. This loads the entire journal into memory.
func (r *runner) discoverFromJournal(
	ctx context.Context,
	maxStreams int,
	seen map[string]struct{},
) (int, error) {
	events, err := r.bundle.Journal.ReadAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("read journal for stream discovery: %w", err)
	}

	return r.collectStreamIDs(events, maxStreams, seen), nil
}

// collectStreamIDs extracts unique stream IDs from events, appending up to
// maxStreams to r.aggIDs/r.refs. Returns the number of events processed.
func (r *runner) collectStreamIDs(
	events []event.Event,
	maxStreams int,
	seen map[string]struct{},
) int {
	for _, evt := range events {
		streamID := evt.StreamID()
		key := streamID.String()

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}

		if len(r.aggIDs) < maxStreams {
			r.aggIDs = append(r.aggIDs, streamID)
			r.refs = append(r.refs, id.NewStreamRef(evt.StreamType(), streamID))
		}
	}

	return len(events)
}
