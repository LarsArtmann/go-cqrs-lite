package projectionhost

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
)

// worker is a single projection's event-processing goroutine.
type worker struct {
	name       string
	projection projection.Projection
	journal    event.SeekableJournal
	cpStore    event.CheckpointStore
	opts       hostOptions
	logger     *slog.Logger

	stateMu sync.RWMutex
	state   WorkerState

	processed atomic.Int64
	errors    atomic.Int64
	restarts  atomic.Int64

	stop chan struct{}
	done chan struct{}
}

func (w *worker) snapshot() WorkerState {
	w.stateMu.RLock()
	defer w.stateMu.RUnlock()

	s := w.state
	s.Processed = w.processed.Load()
	s.Errors = w.errors.Load()
	s.Restarts = int(w.restarts.Load())

	return s
}

func (w *worker) setStatus(s WorkerStatus) {
	w.stateMu.Lock()
	w.state.Status = s
	w.stateMu.Unlock()
}

func (w *worker) setCheckpoint(cp string) {
	w.stateMu.Lock()
	w.state.Checkpoint = cp
	w.stateMu.Unlock()
}

func (w *worker) setLastError(err string) {
	w.stateMu.Lock()
	w.state.LastError = err
	w.stateMu.Unlock()
}

// recordMetric calls fn with the metrics recorder if one is configured.
// Nil-safe; no-op when WithMetrics was not used.
func (w *worker) recordMetric(fn func(MetricsRecorder)) {
	if w.opts.metrics != nil {
		fn(w.opts.metrics)
	}
}

func (w *worker) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(w.done)
	defer w.setStatus(WorkerStopped)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		default:
		}

		w.setStatus(WorkerRunning)

		err := w.process(ctx)
		if err == nil {
			return
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}

		w.errors.Add(1)
		w.setLastError(err.Error())

		restartCount := int(w.restarts.Add(1))
		if w.opts.maxRestarts >= 0 && restartCount > w.opts.maxRestarts {
			w.setStatus(WorkerFailed)
			w.logger.Error("projection worker exhausted restart budget",
				"projection", w.name, "restarts", restartCount, "error", err)

			return
		}

		// Exponential backoff with full jitter: randomize between 0 and the
		// exponential cap so concurrent crashing workers don't all restart at
		// the same instant (thundering herd). math/rand/v2 is auto-seeded.
		exp := min(
			w.opts.backoffInitial*time.Duration(1<<uint(restartCount-1)),
			w.opts.backoffMax,
		)
		backoff := time.Duration(rand.Int64N(int64(exp) + 1))

		w.setStatus(WorkerBackoff)
		w.recordMetric(func(m MetricsRecorder) {
			m.WorkerRestarted(w.name)
		})
		w.logger.Warn("projection worker crashed, restarting after backoff",
			"projection", w.name, "restart", restartCount, "backoff", backoff, "error", err)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		}
	}
}

func (w *worker) process(ctx context.Context) error {
	cp, err := w.cpStore.Load(ctx, w.name)
	if err != nil {
		return err
	}

	var afterID id.EventID
	if !cp.IsZero() {
		afterID = cp.EventID
	}

	w.setCheckpoint(afterID.String())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stop:
			return nil
		default:
		}

		events, err := w.journal.ReadFrom(ctx, afterID, w.opts.batchSize)
		if err != nil {
			return err
		}

		if len(events) == 0 {
			return nil
		}

		for _, evt := range events {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-w.stop:
				return nil
			default:
			}

			if !w.shouldHandle(evt) {
				afterID = evt.ID()

				continue
			}

			start := time.Now()
			err := w.applyWithRetry(ctx, evt)
			duration := time.Since(start)

			if err != nil {
				if w.opts.dlq != nil {
					dlqErr := w.sendToDLQ(ctx, evt, err)
					if dlqErr != nil {
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
				} else {
					return err
				}
			} else {
				w.recordMetric(func(m MetricsRecorder) {
					m.EventProcessed(w.name, string(evt.Type()), duration)
				})
			}

			afterID = evt.ID()

			w.processed.Add(1)
		}

		if err := w.cpStore.Save(ctx, w.name, event.Checkpoint{
			EventID:     afterID,
			ProcessedAt: time.Now(),
		}); err != nil {
			return err
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
			return nil
		}
	}
}

func (w *worker) shouldHandle(evt event.Event) bool {
	types := w.projection.EventTypes()
	if types == nil {
		return true
	}

	evtType := evt.Type()

	return slices.Contains(types, evtType)
}

func (w *worker) applyWithRetry(ctx context.Context, evt event.Event) error {
	var lastErr error

	for attempt := range w.opts.dlqThreshold {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := w.projection.Handle(ctx, evt)
		if err == nil {
			return nil
		}

		lastErr = err

		w.errors.Add(1)
		w.recordMetric(func(m MetricsRecorder) {
			m.EventErrored(w.name, string(evt.Type()))
		})

		// Don't sleep after the final attempt — let the caller decide DLQ.
		if attempt < w.opts.dlqThreshold-1 {
			// Equal jitter: guaranteed minimum of cap/2, plus random up to cap/2.
			// Gives the downstream a real recovery window between per-event
			// retries. Reuses the same backoff params as the restart path.
			exp := min(
				w.opts.backoffInitial*time.Duration(1<<uint(attempt)),
				w.opts.backoffMax,
			)
			half := int64(exp) / 2
			delay := time.Duration(half + rand.Int64N(half+1))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return lastErr
}

func (w *worker) sendToDLQ(ctx context.Context, evt event.Event, handlerErr error) error {
	code, family := "", ""
	if ce, ok := errors.AsType[*event.Error](handlerErr); ok {
		code = ce.Code()
	}

	family = familyToName(event.Classify(handlerErr))

	return w.opts.dlq.Store(ctx, DeadLetterEntry{
		ProjectionName: w.name,
		EventID:        evt.ID().String(),
		EventType:      string(evt.Type()),
		AggregateID:    evt.AggregateID().String(),
		Event:          evt,
		Error:          handlerErr.Error(),
		ErrorCode:      code,
		ErrorFamily:    family,
		FailedAt:       time.Now(),
	})
}

// familyToName maps a taxonomy family to its lowercase wire name.
func familyToName(f event.Family) string {
	switch f {
	case event.Rejection:
		return "rejection"
	case event.Conflict:
		return "conflict"
	case event.Transient:
		return "transient"
	case event.Corruption:
		return "corruption"
	case event.Infrastructure:
		return "infrastructure"
	default:
		return ""
	}
}
