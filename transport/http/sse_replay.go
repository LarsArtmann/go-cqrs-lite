package http

import (
	"context"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

// replayEvents sends missed events to a reconnecting client.
// Events are read from the journal starting after lastEventID and written
// to the client before live streaming begins.
//
// If replayLimit > 0, a single bounded ReadFrom call is used.
// If replayLimit <= 0 (unlimited), events are streamed in batches of
// sseReplayBatchSize so memory stays bounded regardless of journal size.
//
// If broker.replayTimeout > 0, replay is bounded by a context deadline. When
// the deadline fires, replay stops and an SSEReplayIncompleteEvent advisory
// event is written to the client before returning.
//
// Returns a bounded dedup ring of replayed EventIDs for live-phase
// deduplication. The ring covers only the last sseDedupRingCapacity entries,
// which is sufficient because replay→live overlap is always at the tail.
func replayEvents(
	w http.ResponseWriter,
	flusher http.Flusher,
	broker *SSEBroker,
	ctx context.Context,
	lastEventID string,
) *dedupRing {
	if broker.replayTimeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, broker.replayTimeout)
		defer cancel()
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "sse.replay",
		cqrsotel.SpanKindInternal,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString("cqrs.sse.last_event_id", lastEventID),
		),
	)
	defer span.End()

	afterID, err := id.ParseEventID(lastEventID)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil // invalid ID: skip replay, start live
	}

	replayed := newDedupRing(sseDedupRingCapacity)
	totalReplayed := 0
	timedOut := false

	if broker.replayLimit > 0 {
		// Bounded replay: single call with the cap.
		events, err := broker.journal.ReadFrom(ctx, afterID, broker.replayLimit)
		if err != nil {
			if ctx.Err() != nil {
				timedOut = true
			} else {
				cqrsotel.RecordError(span, err)
			}
		} else {
			writeReplayBatch(w, events, replayed)
			totalReplayed = len(events)
		}
	} else {
		// Unlimited replay: stream in batches to keep memory bounded.
		cursor := afterID

		for {
			if ctx.Err() != nil {
				timedOut = true

				break
			}

			events, err := broker.journal.ReadFrom(ctx, cursor, sseReplayBatchSize)
			if err != nil {
				if ctx.Err() != nil {
					timedOut = true
				} else {
					cqrsotel.RecordError(span, err)
				}

				break // journal error or timeout: stop, deliver what we have
			}

			if len(events) == 0 {
				break
			}

			writeReplayBatch(w, events, replayed)
			totalReplayed += len(events)

			cursor = events[len(events)-1].ID()

			if len(events) < sseReplayBatchSize {
				break // journal drained
			}
		}
	}

	if timedOut {
		span.SetAttributes(cqrsotel.AttrString("cqrs.sse.replay_status", "incomplete"))

		_ = WriteSSEEvent(w, SSEEvent{
			Event: SSEReplayIncompleteEvent,
			Data:  `{"message":"replay timed out; some historical events were not delivered"}`,
		})
	}

	flusher.Flush()
	span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, totalReplayed))

	return replayed
}

// writeReplayBatch writes a batch of events to the client and records their
// IDs in the dedup ring.
func writeReplayBatch(w http.ResponseWriter, events []event.Event, replayed *dedupRing) {
	for _, evt := range events {
		replayed.Add(evt.ID().String())

		_ = WriteSSEEvent(w, SSEEvent{
			Event: string(evt.Type()),
			ID:    NewSSEEventID(evt.ID().String()),
			Data:  string(event.PayloadReadOnly(evt)),
		})
	}
}
