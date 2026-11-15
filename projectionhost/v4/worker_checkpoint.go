package projectionhost

import (
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// now returns the current time for checkpoint-cadence decisions, honoring
// an injected clock (WithClock) so interval-cadence tests stay deterministic
// under scheduler delay.
func (w *worker) now() time.Time {
	if w.opts.now != nil {
		return w.opts.now()
	}

	return time.Now()
}

// recordLiveCheckpoint stages the live-phase checkpoint for evt and persists
// it when the configured cadence says so ([WithCheckpointEvery] /
// [WithCheckpointInterval]; default: every event, unchanged from the
// pre-batching behavior). Must be called under handleMu.
func (w *worker) recordLiveCheckpoint(ctx context.Context, evt event.Event) error {
	w.cpPending = event.Checkpoint{
		EventID:     evt.ID(),
		ProcessedAt: w.now(),
	}
	w.cpHasPending = true
	w.cpSinceSave++

	if w.shouldFlushCheckpoint() {
		return w.flushCheckpoint(ctx)
	}

	return nil
}

// shouldFlushCheckpoint reports whether the staged live-phase checkpoint
// should be persisted now: unconditionally by default (every event), or when
// the event-count or time threshold set via the checkpoint options is met.
func (w *worker) shouldFlushCheckpoint() bool {
	if w.opts.cpEvery <= 1 && w.opts.cpInterval <= 0 {
		return true
	}

	if w.opts.cpEvery > 1 && w.cpSinceSave >= w.opts.cpEvery {
		return true
	}

	return w.opts.cpInterval > 0 && w.now().Sub(w.cpLastSave) >= w.opts.cpInterval
}

// flushCheckpoint persists the staged checkpoint if one exists. Must be
// called under handleMu.
func (w *worker) flushCheckpoint(ctx context.Context) error {
	if !w.cpHasPending {
		return nil
	}

	if err := w.cpStore.Save(ctx, w.name, w.cpPending); err != nil {
		return errorfamily.WrapInfrastructure(err, "projectionhost.save_checkpoint_live",
			"save checkpoint after live event")
	}

	w.setCheckpoint(w.cpPending.EventID.String())
	w.cpHasPending = false
	w.cpSinceSave = 0
	w.cpLastSave = w.now()

	return nil
}

// flushPendingCheckpoint persists any staged live-phase checkpoint at worker
// shutdown, so a graceful Stop does not widen the reprocessing window beyond
// the configured cadence. ctx should be detached from the worker's lifecycle
// (e.g. context.WithoutCancel): the worker's own context is already cancelled
// by the time this runs.
func (w *worker) flushPendingCheckpoint(ctx context.Context) {
	w.handleMu.Lock()
	defer w.handleMu.Unlock()

	if !w.cpHasPending {
		return
	}

	if err := w.flushCheckpoint(ctx); err != nil {
		w.logger.Warn("flush pending live checkpoint at shutdown failed",
			"projection", w.name, "error", err)
	}
}
