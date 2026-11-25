package projectionhost

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// liveSubscription is one registration of a worker's live handler with the
// subscriber, plus the latch that retires it. event.Subscriber has no
// unsubscribe, so a registered handler outlives the live phase that created
// it: without the latch every restart left the previous handler active, and
// a worker that exhausted its restart budget kept applying events while
// reporting WorkerFailed.
type liveSubscription struct {
	active atomic.Bool
}

func newLiveSubscription() *liveSubscription {
	s := &liveSubscription{} //nolint:exhaustruct_v5 // armed below
	s.active.Store(true)

	return s
}

// retire turns the registered handler into a no-op. Called when the live
// phase that registered it ends in an error; the journal keeps every event
// the retired handler skips, so the next live phase's catch-up drain (or the
// next Start) picks them up from the checkpoint.
func (s *liveSubscription) retire() { s.active.Store(false) }

// handler returns the event.Handler for the live subscriber.
// The handler acquires handleMu to serialize with any concurrent catch-up
// drain processing, and marks applied events as seen so the subsequent (or
// concurrent) catch-up drain skips them instead of double-applying.
func (s *liveSubscription) handler(w *worker, ctx context.Context) event.Handler {
	return func(_ context.Context, evt event.Event) error {
		if !s.active.Load() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stop:
			return nil
		default:
		}

		w.handleMu.Lock()
		defer w.handleMu.Unlock()

		if w.wasSeen(evt.ID()) {
			return nil
		}

		if !w.shouldHandle(evt) {
			w.markSeen(evt.ID())
			w.lastProcessedNs.Store(time.Now().UnixNano())

			return nil
		}

		start := time.Now()
		handleErr := w.applyWithRetry(ctx, evt)
		duration := time.Since(start)

		if handleErr != nil {
			if e := w.handleProcessEventError(ctx, evt, handleErr); e != nil {
				return e
			}
		} else {
			w.recordMetric(func(m MetricsRecorder) {
				m.EventProcessed(w.name, string(evt.Type()), duration)
			})
		}

		if saveErr := w.recordLiveCheckpoint(ctx, evt); saveErr != nil {
			w.warnLiveCheckpoint(evt, saveErr)
		}

		w.markSeen(evt.ID())
		w.processed.Add(1)
		w.lastProcessedNs.Store(time.Now().UnixNano())

		return nil
	}
}

// warnLiveCheckpoint reports a failed live-phase checkpoint save. The event
// is already journaled and applied; the checkpoint is bookkeeping that only
// bounds reprocessing. Failing the handler here would fail the publisher's
// dispatch (in-process buses run handlers synchronously; EventBus sets
// BlockPublishUntilSubscriberAck) for a write that IS durable.
// recordLiveCheckpoint staged the checkpoint before flushing, so cpHasPending
// stays set: the next live event retries the save and Stop's
// flushPendingCheckpoint gives the final retry (issue #128).
func (w *worker) warnLiveCheckpoint(evt event.Event, saveErr error) {
	w.logger.Warn("live checkpoint save failed; will retry on next event",
		"projection", w.name,
		"event_id", evt.ID().String(),
		"error", saveErr)
}
