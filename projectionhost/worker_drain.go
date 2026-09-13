package projectionhost

import (
	"context"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// process drains the journal from the last checkpoint. When caught up (ReadFrom
// returns zero events), it transitions to live subscription if a subscriber is
// configured.
func (w *worker) process(ctx context.Context) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projectionhost.drain",
		cqrsotel.SpanKindInternal,
		cqrsotel.WithAttributes(cqrsotel.AttrString("cqrs.projection.name", w.name)),
	)
	defer span.End()

	checkpoint, err := w.cpStore.Load(ctx, w.name)
	if err != nil {
		return fmt.Errorf("load checkpoint for %q: %w", w.name, err)
	}

	var afterID id.EventID
	if !checkpoint.IsZero() {
		afterID = checkpoint.EventID
	}

	w.setCheckpoint(afterID.String())

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("drain cancelled: %w", ctx.Err())
		case <-w.stop:
			return nil
		default:
		}

		events, err := w.journal.ReadFrom(ctx, afterID, w.opts.batchSize)
		if err != nil {
			return fmt.Errorf("read journal batch: %w", err)
		}

		if len(events) == 0 {
			break
		}

		for _, evt := range events {
			select {
			case <-ctx.Done():
				return fmt.Errorf("drain event loop cancelled: %w", ctx.Err())
			case <-w.stop:
				return nil
			default:
			}

			if err := w.processEvent(ctx, evt); err != nil {
				return err
			}

			afterID = evt.ID()
		}

		if err := w.cpStore.Save(ctx, w.name, event.Checkpoint{
			EventID:     afterID,
			ProcessedAt: time.Now(),
		}); err != nil {
			return fmt.Errorf("save checkpoint: %w", err)
		}

		w.setCheckpoint(afterID.String())

		// Report checkpoint lag: how far behind real-time the projection is.
		if len(events) > 0 {
			last := events[len(events)-1]

			w.recordMetric(func(m MetricsRecorder) {
				m.CheckpointAdvanced(w.name, time.Since(last.OccurredAt()))
			})
		}

		if len(events) < w.opts.batchSize {
			break
		}
	}

	// Phase 2: live subscription (if configured).
	if w.opts.subscriber != nil {
		return w.processLive(ctx, afterID)
	}

	return nil
}

// handleProcessEventError handles an error from applyWithRetry. If a DLQ is
// configured, the event is sent to the dead-letter store and nil is returned
// (the error is swallowed — the event is quarantined, not fatal). If no DLQ
// is configured, the original error is returned (fatal).
func (w *worker) handleProcessEventError(
	ctx context.Context,
	evt event.Event,
	err error,
) error {
	if w.opts.dlq == nil {
		return err
	}

	// Only non-retryable families (Rejection/Corruption) park in the DLQ.
	// Parking a Transient/Infrastructure failure would quarantine an event
	// that a later retry could succeed on — a permanent silent gap until
	// someone replays by hand. Retryable failures return to the caller
	// instead: the checkpoint stays put, the worker restarts with backoff,
	// and the journal re-delivers the event. If restarts exhaust, the
	// worker fails loudly (WorkerFailed + onFailed) instead of skipping.
	if errorfamily.IsRetryable(err) {
		return err
	}

	if dlqErr := w.sendToDLQ(ctx, evt, err); dlqErr != nil {
		return dlqErr
	}

	w.recordMetric(func(m MetricsRecorder) {
		m.EventDeadLettered(w.name, string(evt.Type()))
	})

	w.logger.Warn("event sent to dead-letter queue after retries",
		"projection", w.name,
		"event_id", evt.ID().String(),
		"event_type", string(evt.Type()),
		"error", err)

	return nil
}

// processLive subscribes to live events via the configured subscriber, then
// performs a catch-up drain to close the TOCTOU race window between the initial
// journal drain and subscriber registration.
//
// WorkerLive is set BEFORE SubscribeAll because SubscribeAll IS the live phase:
// for blocking subscribers (NATS, Postgres LISTEN/NOTIFY, Watermill GoChannel),
// the call blocks for the entire lifetime of the projection while delivering
// events via the callback. Setting WorkerLive after would mean it is never
// visible during operation.
//
// For non-blocking subscribers (in-process buses like simpleBus), SubscribeAll
// returns immediately after registering the callback. The catch-up drain then
// reads events published between the initial drain and subscription — closing
// the race window where events could be silently lost.
//
// The handleMu mutex serializes event processing between the catch-up drain
// and the live handler callback, preventing concurrent projection.Handle calls.
func (w *worker) processLive(ctx context.Context, afterID id.EventID) error {
	w.setStatus(WorkerLive)

	//cqrs-lint:ignore(A005,C027) library code or intentional pattern
	if err := w.opts.subscriber.SubscribeAll(w.liveHandler(ctx)); err != nil {
		return fmt.Errorf("subscribe live events: %w", err)
	}

	if err := w.drainCatchUp(ctx, afterID); err != nil {
		return fmt.Errorf("catch-up drain: %w", err)
	}

	return nil
}

// liveHandler returns the event.Handler callback for the live subscriber.
// The handler acquires handleMu to serialize with any concurrent catch-up
// drain processing.
func (w *worker) liveHandler(ctx context.Context) event.Handler {
	return func(_ context.Context, evt event.Event) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stop:
			return nil
		default:
		}

		w.handleMu.Lock()
		defer w.handleMu.Unlock()

		if w.wasSeen(evt.ID().String()) {
			return nil
		}

		if !w.shouldHandle(evt) {
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
			return saveErr
		}

		w.processed.Add(1)
		w.lastProcessedNs.Store(time.Now().UnixNano())

		return nil
	}
}

// processEvent applies a single event to the projection (with retry), recording
// metrics and updating internal counters. Events that don't match the
// projection's filter are skipped (marked as seen but not processed). Returns
// a fatal error if the event cannot be processed even after retries and DLQ
// routing. Shared by the initial drain and the catch-up drain.
func (w *worker) processEvent(ctx context.Context, evt event.Event) error {
	if !w.shouldHandle(evt) {
		w.markSeen(evt.ID().String())

		return nil
	}

	start := time.Now()
	err := w.applyWithRetry(ctx, evt)
	duration := time.Since(start)

	if err != nil {
		if e := w.handleProcessEventError(ctx, evt, err); e != nil {
			return e
		}
	} else {
		w.recordMetric(func(m MetricsRecorder) {
			m.EventProcessed(w.name, string(evt.Type()), duration)
		})
	}

	w.processed.Add(1)
	w.markSeen(evt.ID().String())
	w.lastProcessedNs.Store(time.Now().UnixNano())

	return nil
}

// drainCatchUp reads events from the journal starting at afterID, processing
// any that were published between the initial drain completing and the live
// subscriber being registered. This closes the TOCTOU race window where events
// could be silently lost with non-blocking subscribers.
//
// Each event is processed under handleMu to serialize with concurrent live
// handler callbacks. The seenIDs ring prevents double-processing at the overlap.
func (w *worker) drainCatchUp(ctx context.Context, afterID id.EventID) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.stop:
			return nil
		default:
		}

		events, err := w.journal.ReadFrom(ctx, afterID, w.opts.batchSize)
		if err != nil {
			return fmt.Errorf("read journal batch: %w", err)
		}

		if len(events) == 0 {
			return nil
		}

		for _, evt := range events {
			select {
			case <-ctx.Done():
				return nil
			case <-w.stop:
				return nil
			default:
			}

			w.handleMu.Lock()
			perr := w.processEvent(ctx, evt)
			w.handleMu.Unlock()

			if perr != nil {
				return perr
			}

			afterID = evt.ID()
		}

		if err := w.cpStore.Save(ctx, w.name, event.Checkpoint{
			EventID:     afterID,
			ProcessedAt: time.Now(),
		}); err != nil {
			return fmt.Errorf("save checkpoint: %w", err)
		}

		w.setCheckpoint(afterID.String())

		if len(events) < w.opts.batchSize {
			return nil
		}
	}
}
