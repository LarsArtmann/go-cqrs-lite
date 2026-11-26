package irohengine

import (
	"context"
	"errors"
	"fmt"
	"sync"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Close stops replication and closes the local engine.
func (e *replicatedEngine) Close() error {
	if e.cfg.transport != nil {
		if err := e.cfg.transport.Close(); err != nil {
			// The local engine is still closed: a transport that fails to
			// shut down must not leak it.
			return errors.Join(err, e.local.Close())
		}
	}

	return e.local.Close()
}

// HealthCheck verifies the replication stack is responsive: the local engine
// (when it implements [metaengine.HealthChecker]) and the transport (when it
// implements [LivenessReporter]). A transport with no liveness report is
// assumed healthy — there is no application-data-free probe for it.
//
// A failed transport publish is also reported (until a later publish
// succeeds): writes apply locally first, so this is the only place a write
// that never reached the peers becomes visible.
// Implements [metaengine.HealthChecker].
func (e *replicatedEngine) HealthCheck(ctx context.Context) error {
	if hc, ok := e.local.(metaengine.HealthChecker); ok {
		if err := hc.HealthCheck(ctx); err != nil {
			return fmt.Errorf("iroh: local engine: %w", err)
		}
	}

	if e.cfg.transport != nil {
		if lr, ok := e.cfg.transport.(LivenessReporter); ok {
			if err := lr.Healthy(ctx); err != nil {
				return fmt.Errorf("iroh: transport: %w", err)
			}
		}
	}

	if err := e.pubErr.last(); err != nil {
		return fmt.Errorf("iroh: transport: %w", err)
	}

	return nil
}

// publish sends a WriteOp to remote nodes. No re-entrancy guard needed:
// applyRemote calls the local engine directly, never the wrapper methods,
// so publish cannot be triggered from a remote-op application.
//
// The write is already applied locally when this runs, so a failed publish is
// not returned to the caller (a retry would apply the write twice); it is
// kept for HealthCheck instead.
func (e *replicatedEngine) publish(ctx context.Context, op WriteOp) {
	if e.cfg.transport == nil {
		return
	}

	op.ID = nextOpID()
	op.PublishedAt = e.cfg.clock.Now()

	e.pubErr.record(e.cfg.transport.Publish(ctx, op))
}

// publishHealth remembers the outcome of the most recent transport publish.
// Separate from e.mu so HealthCheck never waits behind a remote op being
// applied to the local engine.
type publishHealth struct {
	mu     sync.Mutex
	failed error
}

// record stores err as the latest publish outcome; nil clears a failure.
func (h *publishHealth) record(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.failed = err
}

// last returns the failure of the most recent publish, or nil.
func (h *publishHealth) last() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.failed
}
