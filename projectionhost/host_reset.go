package projectionhost

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Resettable is an optional interface a Projection can implement to support
// full state reset. When Host.Reset detects that the projection implements
// Resettable, it calls Reset(ctx) so the projection can clear its read-model
// state (e.g. drop all rows from a SQL table, flush a KV store).
//
// Projections that don't implement Resettable will only have their checkpoint
// dropped — the next Start replays from zero, but any stale read-model state
// remains unless the consumer clears it manually.
type Resettable interface {
	Reset(ctx context.Context) error
}

// ResetOption configures a Reset call.
type ResetOption func(*resetConfig)

type resetConfig struct {
	purgeDLQ       bool
	keepStaleState bool
}

// WithPurgeDeadLetters instructs Reset to also purge dead-letter entries for
// the named projection from the DeadLetterStore (if one is configured). Use this
// when rebuilding a projection from scratch: stale poison entries from a
// previous handler bug serve no purpose once the projection is wiped.
//
//	host.Reset(ctx, "users", projectionhost.WithPurgeDeadLetters())
func WithPurgeDeadLetters() ResetOption {
	return func(c *resetConfig) { c.purgeDLQ = true }
}

// WithKeepStaleState acknowledges that a Reset of a projection which does NOT
// implement [Resettable] clears only its checkpoint, leaving any existing
// read-model state in place for the replay to be applied on top of. Passing
// this option silences the warning that Reset otherwise logs for such a
// partial reset.
//
// Use it when the projection's handler is idempotent (re-applying events over
// stale rows is a no-op) or when the read-model state is cleared out-of-band.
// Without it — and without a [Resettable] implementation — Reset logs a warning
// so a partial revert is never silent. In v5 this warning becomes a hard error
// (rides the ADR-0123 composition-root wave); WithKeepStaleState is the
// permanent opt-out that keeps the checkpoint-only reset working.
//
//	host.Reset(ctx, "users", projectionhost.WithKeepStaleState())
func WithKeepStaleState() ResetOption {
	return func(c *resetConfig) { c.keepStaleState = true }
}

// Reset drops the checkpoint for the named projection and, if the projection
// implements [Resettable], calls its Reset method to clear read-model state.
// After Reset, the next Start replays all events from the beginning of the
// journal. Use this to rebuild a projection from scratch after fixing a
// handler bug.
//
// Pass [WithPurgeDeadLetters] to also clear dead-letter entries for the
// projection from the configured DeadLetterStore.
//
// If the projection does NOT implement [Resettable], Reset clears only the
// checkpoint and logs a warning that stale read-model state remains (a partial
// revert). Pass [WithKeepStaleState] to acknowledge that intent and silence the
// warning. In v5 the warning becomes a hard error.
//
// Reset returns an error if the projection name is not registered or the host
// is currently running (Stop first). It is safe to call Reset multiple times.
func (h *Host) Reset(ctx context.Context, name string, opts ...ResetOption) error {
	cfg := resetConfig{purgeDLQ: false, keepStaleState: false}
	for _, opt := range opts {
		opt(&cfg)
	}

	h.mu.Lock()

	if h.started && !h.stopped {
		h.mu.Unlock()

		return errorfamily.NewRejection(
			"projectionhost.reset_while_running",
			"projectionhost: cannot reset while host is running — Stop first",
		)
	}

	w, ok := h.workers[name]
	if !ok {
		h.mu.Unlock()

		return errorfamily.NewRejection(
			"projectionhost.unknown_projection",
			fmt.Sprintf("projection %q is not registered", name),
		)
	}

	h.mu.Unlock()

	// Checkpoint first: if the process dies between the two writes, the
	// next Start replays from zero ON TOP of stale read-model state, which
	// idempotent handlers absorb. The reverse order is unsafe — read-model
	// cleared but checkpoint alive means pre-checkpoint events are skipped
	// and never re-projected.
	if err := h.cpStore.Save(
		ctx,
		name,
		event.Checkpoint{}, //nolint:exhaustruct_v5 // zero-value is the cleared-checkpoint intent
	); err != nil {
		return errorfamily.WrapInfrastructure(err, "projectionhost.reset_checkpoint",
			fmt.Sprintf("clear checkpoint for %q", name))
	}

	if r, ok := w.projection.(Resettable); ok {
		if err := r.Reset(ctx); err != nil {
			return errorfamily.WrapInfrastructure(err, "projectionhost.reset_projection",
				fmt.Sprintf("reset projection %q", name))
		}
	} else if !cfg.keepStaleState {
		h.opts.logger.Warn("projectionhost: Reset cleared the checkpoint but the projection does not implement Resettable, so stale read-model state remains and the replay will re-apply events on top of it",
			"projection", name,
			"remedy", "implement projectionhost.Resettable to clear the read-model state, or pass projectionhost.WithKeepStaleState() to acknowledge the checkpoint-only reset and silence this warning",
		)
	}

	w.setCheckpoint("")

	if cfg.purgeDLQ && h.opts.dlq != nil {
		if err := h.opts.dlq.Purge(ctx, name); err != nil {
			return errorfamily.WrapInfrastructure(err, "projectionhost.reset_dlq_purge",
				fmt.Sprintf("purge dead-letter entries for %q", name))
		}
	}

	return nil
}
