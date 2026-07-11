package http

import (
	"context"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
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

	ringCap := broker.dedupRingCap
	if ringCap <= 0 {
		ringCap = sseDedupRingCapacity
	}

	replayed := dedup.NewRing(ringCap)
	start := time.Now()

	budget := resolveReplayBudget(broker)

	res := runReplayLoop(ctx, broker, w, afterID, replayed, budget)
	if res.journalErr != nil {
		cqrsotel.RecordError(span, res.journalErr)
	}

	if res.timedOut || res.budgetExceeded {
		status := "incomplete"
		if res.budgetExceeded {
			status = "byte_budget_exceeded"
		}

		span.SetAttributes(cqrsotel.AttrString("cqrs.sse.replay_status", status))
		span.SetAttributes(cqrsotel.AttrInt("cqrs.sse.replay_bytes", res.totalBytes))

		_ = WriteSSEEvent(w, SSEEvent{
			Event: SSEReplayIncompleteEvent,
			Data:  `{"message":"replay stopped early; some historical events were not delivered"}`,
		})
	}

	flusher.Flush()
	span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, res.totalReplayed))
	span.SetAttributes(cqrsotel.AttrInt("cqrs.sse.dedup_ring_size", replayed.Len()))

	durationMs := float64(
		time.Since(start).Microseconds(),
	) / float64(
		time.Millisecond/time.Microsecond,
	)

	broker.replayMetrics.RecordReplay(
		ctx,
		durationMs,
		res.totalReplayed,
		res.timedOut || res.budgetExceeded,
	)

	return replayed
}

// replayResult holds the mutable state accumulated during a replay pass.
type replayResult struct {
	totalReplayed  int
	totalBytes     int
	timedOut       bool
	budgetExceeded bool
	journalErr     error
}

// resolveReplayBudget applies the default byte budget for unlimited replay when
// no explicit budget was set (zero value). Bounded replay (replayLimit > 0) is
// already capped by event count. SSEReplayBudgetDisabled (-1) skips the
// auto-default for truly unlimited replay.
func resolveReplayBudget(broker *SSEBroker) int {
	if broker.replayLimit <= 0 && broker.replayByteBudget == 0 {
		return sseDefaultReplayByteBudget
	}

	return broker.replayByteBudget
}

// runReplayLoop dispatches to the bounded or unlimited replay path.
func runReplayLoop(
	ctx context.Context,
	broker *SSEBroker,
	w http.ResponseWriter,
	afterID id.EventID,
	replayed *dedup.Ring,
	budget int,
) replayResult {
	if broker.replayLimit > 0 {
		return runBoundedReplay(ctx, broker, w, afterID, replayed, budget)
	}

	return runUnlimitedReplay(ctx, broker, w, afterID, replayed, budget)
}

// runBoundedReplay reads events in a single bounded call.
func runBoundedReplay(
	ctx context.Context,
	broker *SSEBroker,
	w http.ResponseWriter,
	afterID id.EventID,
	replayed *dedup.Ring,
	budget int,
) replayResult {
	var res replayResult

	events, err := broker.journal.ReadFrom(ctx, afterID, broker.replayLimit)
	if err != nil {
		if ctx.Err() != nil {
			res.timedOut = true
		} else {
			res.journalErr = err
		}

		return res
	}

	written, count, budgetHit := writeReplayBatchBounded(
		w, events, replayed, 0, budget, broker.payloadTransform,
	)
	res.totalBytes = written
	res.totalReplayed = count
	res.budgetExceeded = budgetHit

	return res
}

// runUnlimitedReplay streams events in batches to keep memory bounded.
func runUnlimitedReplay(
	ctx context.Context,
	broker *SSEBroker,
	w http.ResponseWriter,
	afterID id.EventID,
	replayed *dedup.Ring,
	budget int,
) replayResult {
	var res replayResult

	cursor := afterID

	for {
		if ctx.Err() != nil {
			res.timedOut = true

			return res
		}

		events, err := broker.journal.ReadFrom(ctx, cursor, sseReplayBatchSize)
		if err != nil {
			if ctx.Err() != nil {
				res.timedOut = true
			} else {
				res.journalErr = err
			}

			return res
		}

		if len(events) == 0 {
			return res
		}

		written, count, budgetHit := writeReplayBatchBounded(
			w, events, replayed, res.totalBytes, budget, broker.payloadTransform,
		)
		res.totalBytes += written
		res.totalReplayed += count

		if budgetHit {
			res.budgetExceeded = true

			return res
		}

		cursor = events[len(events)-1].ID()

		if len(events) < sseReplayBatchSize {
			return res // journal drained
		}
	}
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
	transform func(event.Event) []byte,
) (int, int, bool) {
	var bytesWritten, eventsWritten int

	for _, evt := range events {
		var data string
		if transform != nil {
			data = string(transform(evt))
		} else {
			data = string(event.PayloadReadOnly(evt))
		}

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
