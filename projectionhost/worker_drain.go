package projectionhost

import (
	"context"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
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

	cp, err := w.cpStore.Load(ctx, w.name)
	if err != nil {
		return fmt.Errorf("load checkpoint for %q: %w", w.name, err)
	}

	var afterID id.EventID
	if !cp.IsZero() {
		afterID = cp.EventID
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

			if !w.shouldHandle(evt) {
				afterID = evt.ID()
				w.markSeen(evt.ID().String())

				continue
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

			afterID = evt.ID()

			w.processed.Add(1)
			w.markSeen(evt.ID().String())
			w.lastProcessedNs.Store(time.Now().UnixNano())
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
		return w.processLive(ctx)
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

// processLive subscribes to live events via the configured subscriber. Events
// already processed during journal drain are silently skipped. Blocks until the
// context is cancelled, the subscriber returns an error, or the worker is stopped.
func (w *worker) processLive(ctx context.Context) error {
	w.setStatus(WorkerLive)

	if err := w.opts.subscriber.SubscribeAll(func(_ context.Context, evt event.Event) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stop:
			return nil
		default:
		}

		// Dedup: skip events that were already processed during journal drain.
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

		if saveErr := w.cpStore.Save(ctx, w.name, event.Checkpoint{
			EventID:     evt.ID(),
			ProcessedAt: time.Now(),
		}); saveErr != nil {
			return errorfamily.WrapInfrastructure(saveErr, "projectionhost.save_checkpoint_live",
				"save checkpoint after live event")
		}

		w.processed.Add(1)
		w.lastProcessedNs.Store(time.Now().UnixNano())

		return nil
	}); err != nil {
		return fmt.Errorf("subscribe live events: %w", err)
	}

	return nil
}
