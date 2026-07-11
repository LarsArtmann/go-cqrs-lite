package projectionhost

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/dedup/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
)

// jitterHalfDivisor splits the backoff window in half so jitter is symmetric
// around the midpoint (delay = half + rand(0..half]).
const jitterHalfDivisor = 2

// worker is a single projection's event-processing goroutine.
type worker struct {
	name       string
	projection projection.Projection
	journal    event.SeekableJournal
	cpStore    event.CheckpointStore
	opts       hostOptions
	logger     *slog.Logger

	// typeSet is a pre-built set of event types the projection handles.
	// nil means "all events". Built once at construction to make shouldHandle O(1).
	typeSet map[event.Type]struct{}

	stateMu sync.RWMutex
	state   WorkerState

	processed       atomic.Int64
	errors          atomic.Int64
	restarts        atomic.Int64
	lastProcessedNs atomic.Int64 // Unix nanoseconds of the most recently processed event

	// seenIDs is a bounded ring of event IDs accumulated during journal drain
	// so the live phase can skip events that overlap the replay→live boundary.
	// Bounded to dedup.DefaultCapacity entries — never grows during live
	// processing. Not safe for concurrent use; only accessed during drain
	// (single goroutine).
	seenIDs *dedup.Ring

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
	s.Lag = w.lagDuration()

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
	defer func() {
		if w.snapshot().Status != WorkerFailed {
			w.setStatus(WorkerStopped)
		}
	}()

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
			w.recordMetric(func(m MetricsRecorder) {
				m.WorkerFailed(w.name)
			})

			if w.opts.onFailed != nil {
				w.opts.onFailed(w.name, err.Error())
			}

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
		backoff := time.Duration(
			rand.Int64N(int64(exp) + 1), //nolint:gosec // non-crypto backoff jitter
		)

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

func (w *worker) shouldHandle(evt event.Event) bool {
	if w.typeSet == nil {
		return true
	}

	_, ok := w.typeSet[evt.Type()]

	return ok
}

func (w *worker) applyWithRetry(ctx context.Context, evt event.Event) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "projectionhost.handle_event",
		cqrsotel.SpanKindConsumer,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString("cqrs.projection.name", w.name),
			cqrsotel.AttrString("cqrs.event.type", string(evt.Type())),
			cqrsotel.AttrString("cqrs.event.id", evt.ID().String()),
		),
	)
	defer span.End()

	var lastErr error

	for attempt := range w.opts.dlqThreshold {
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
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
			half := int64(exp) / jitterHalfDivisor
			delay := time.Duration(
				half + rand.Int64N(half+1), //nolint:gosec // non-crypto backoff jitter
			)

			select {
			case <-ctx.Done():
				return fmt.Errorf("backoff cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	cqrsotel.RecordError(span, lastErr)

	return lastErr
}

func (w *worker) sendToDLQ(ctx context.Context, evt event.Event, handlerErr error) error {
	var code string

	if ce, ok := errors.AsType[*errorfamily.Error](handlerErr); ok {
		code = ce.Code()
	}

	family := familyToName(errorfamily.Classify(handlerErr))

	if err := w.opts.dlq.Store(ctx, DeadLetterEntry{
		ProjectionName: w.name,
		EventID:        evt.ID().String(),
		EventType:      string(evt.Type()),
		AggregateID:    evt.AggregateID().String(),
		Event:          evt,
		Error:          handlerErr.Error(),
		ErrorCode:      code,
		ErrorFamily:    family,
		FailedAt:       time.Now(),
	}); err != nil {
		return fmt.Errorf("store dead-letter entry: %w", err)
	}

	return nil
}

// familyToName maps a taxonomy family to its lowercase wire name.
func familyToName(f errorfamily.Family) string {
	switch f {
	case errorfamily.Rejection:
		return "rejection"
	case errorfamily.Conflict:
		return "conflict"
	case errorfamily.Transient:
		return "transient"
	case errorfamily.Corruption:
		return "corruption"
	case errorfamily.Infrastructure:
		return "infrastructure"
	default:
		return ""
	}
}

// markSeen records an event ID as processed during journal drain.
func (w *worker) markSeen(id string) {
	w.seenIDs.Add(id)
}

// wasSeen reports whether an event ID was seen during journal drain.
func (w *worker) wasSeen(id string) bool {
	return w.seenIDs.Has(id)
}

// lastProcessedAt returns the wall-clock time of the most recently processed
// event, or the zero time if the worker has not processed any event yet.
func (w *worker) lastProcessedAt() time.Time {
	nanos := w.lastProcessedNs.Load()
	if nanos == 0 {
		return time.Time{}
	}

	return time.Unix(0, nanos)
}

// lagDuration returns how long since the most recently processed event for this
// worker. Returns 0 if the worker has not processed any event yet.
func (w *worker) lagDuration() time.Duration {
	ts := w.lastProcessedAt()
	if ts.IsZero() {
		return 0
	}

	return time.Since(ts)
}
