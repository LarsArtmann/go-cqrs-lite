package http

import (
	"context"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/dedup/v3"
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
) *dedup.Ring {
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

	replayed := dedup.NewRing(broker.dedupRingCap) // falls back to default when <=0
	totalReplayed := 0
	totalBytes := 0
	timedOut := false
	byteBudgetExceeded := false
	start := time.Now()

	budget := broker.replayByteBudget // <=0 = disabled (count-based batching only)

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
			written, count, budgetHit := writeReplayBatchBounded(
				w,
				events,
				replayed,
				totalBytes,
				budget,
			)
			totalBytes += written
			totalReplayed += count

			if budgetHit {
				byteBudgetExceeded = true
			}
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

			written, count, budgetHit := writeReplayBatchBounded(
				w,
				events,
				replayed,
				totalBytes,
				budget,
			)
			totalBytes += written
			totalReplayed += count

			if budgetHit {
				byteBudgetExceeded = true

				break
			}

			cursor = events[len(events)-1].ID()

			if len(events) < sseReplayBatchSize {
				break // journal drained
			}
		}
	}

	if timedOut || byteBudgetExceeded {
		status := "incomplete"
		if byteBudgetExceeded {
			status = "byte_budget_exceeded"
		}

		span.SetAttributes(cqrsotel.AttrString("cqrs.sse.replay_status", status))
		span.SetAttributes(cqrsotel.AttrInt("cqrs.sse.replay_bytes", totalBytes))

		_ = WriteSSEEvent(w, SSEEvent{
			Event: SSEReplayIncompleteEvent,
			Data:  `{"message":"replay stopped early; some historical events were not delivered"}`,
		})
	}

	flusher.Flush()
	span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, totalReplayed))

	broker.replayMetrics.RecordReplay(
		ctx,
		float64(time.Since(start).Microseconds())/1000.0,
		totalReplayed,
		timedOut || byteBudgetExceeded,
	)

	return replayed
}

// writeReplayBatchBounded writes events to the client, recording IDs in the
// dedup ring. If budget > 0 and the cumulative payload bytes (priorBytes +
// this batch) would exceed budget, writing stops mid-batch and budgetHit is
// returned true. Returns (bytesWritten, eventsWritten, budgetHit).
//
// When budget <= 0, all events are written (byte-budgeting disabled).
func writeReplayBatchBounded(
	w http.ResponseWriter,
	events []event.Event,
	replayed *dedup.Ring,
	priorBytes, budget int,
) (bytesWritten, eventsWritten int, budgetHit bool) {
	for _, evt := range events {
		data := string(event.PayloadReadOnly(evt))

		if budget > 0 && priorBytes+bytesWritten+len(data) > budget {
			return bytesWritten, eventsWritten, true
		}

		replayed.Add(evt.ID().String())

		bytesWritten += len(data)
		eventsWritten++

		_ = WriteSSEEvent(w, SSEEvent{
			Event: string(evt.Type()),
			ID:    NewSSEEventID(evt.ID().String()),
			Data:  data,
		})
	}

	return bytesWritten, eventsWritten, false
}
