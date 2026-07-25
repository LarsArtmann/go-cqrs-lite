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
	purgeDLQ bool
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

// Reset drops the checkpoint for the named projection and, if the projection
// implements [Resettable], calls its Reset method to clear read-model state.
// After Reset, the next Start replys all events from the beginning of the
// journal. Use this to rebuild a projection from scratch after fixing a
// handler bug.
//
// Pass [WithPurgeDeadLetters] to also clear dead-letter entries for the
// projection from the configured DeadLetterStore.
//
// Reset returns an error if the projection name is not registered or the host
// is currently running (Stop first). It is safe to call Reset multiple times.
func (h *Host) Reset(ctx context.Context, name string, opts ...ResetOption) error {
	cfg := resetConfig{purgeDLQ: false}
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

	if r, ok := w.projection.(Resettable); ok {
		if err := r.Reset(ctx); err != nil {
			return errorfamily.WrapInfrastructure(err, "projectionhost.reset_projection",
				fmt.Sprintf("reset projection %q", name))
		}
	}

	if err := h.cpStore.Save(
		ctx,
		name,
		event.Checkpoint{}, //nolint:exhaustruct // zero-value is the cleared-checkpoint intent
	); err != nil {
		return errorfamily.WrapInfrastructure(err, "projectionhost.reset_checkpoint",
			fmt.Sprintf("clear checkpoint for %q", name))
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
