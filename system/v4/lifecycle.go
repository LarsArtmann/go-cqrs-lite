package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// CommandLifecycleResult bundles everything needed to wire ADR-0117 command
// lifecycle tracking into a [DomainConfig].
type CommandLifecycleResult struct {
	// Recorder is the lifecycle event recorder. Keep a reference if you need
	// to emit lifecycle events manually (e.g. from a custom dead-letter handler).
	Recorder *commandlifecycle.Recorder

	// OuterMiddleware emits command.received, command.completed, and
	// command.dead-lettered. Place it FIRST in DomainConfig.Middleware, before
	// any retry middleware.
	OuterMiddleware command.Middleware

	// AttemptMiddleware emits command.failed and command.retried per attempt.
	// Place it AFTER your retry middleware in DomainConfig.Middleware.
	AttemptMiddleware command.Middleware

	// Projections are the pre-built lifecycle projection declarations (DLQ,
	// retry count, failure log, rejection log, processing time, commands by
	// actor) ready for DomainConfig.Projections.
	Projections []ProjectionDeclaration
}

// WithCommandLifecycle creates a [commandlifecycle.Recorder] and returns the
// middleware pair plus pre-built projection declarations for one-call wiring
// into a [DomainConfig].
//
// The store must implement [event.Store] (both [event.EventSink] and
// [event.EventSource]) so the Recorder can derive stream versions from the
// existing event log and survive restarts.
//
// Example:
//
//	cl := system.WithCommandLifecycle(eventStore)
//	config := system.DomainConfig{
//	    Middleware:  []command.Middleware{
//	        cl.OuterMiddleware,
//	        middleware.CommandRetry(retryCfg),
//	        cl.AttemptMiddleware,
//	    },
//	    Projections: cl.Projections,
//	}
func WithCommandLifecycle(
	store event.Store,
	opts ...commandlifecycle.RecorderOption,
) CommandLifecycleResult {
	recorder := commandlifecycle.NewRecorder(store, opts...)
	outer, attempt := commandlifecycle.New(recorder)

	projs := make([]ProjectionDeclaration, 0, len(projections.All()))
	for _, decl := range projections.All() {
		projs = append(projs, RawQuery(decl))
	}

	return CommandLifecycleResult{
		Recorder:          recorder,
		OuterMiddleware:   outer,
		AttemptMiddleware: attempt,
		Projections:       projs,
	}
}

// Close shuts down all owned infrastructure: managed timers (stopped and
// waited), projection host, engines, stores. All close errors are joined and
// returned (not just the first), matching [stack.Bundle.Close] behavior.
func (s *System) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}

	s.stopped = true

	// Stop managed timers BEFORE any teardown: scheduler dispatch must not
	// race the engines and buses being closed underneath it. GracefulClose
	// already stopped them (Phase 0); stopTimersLocked is idempotent.
	s.stopTimersLocked()

	var errs []error

	if s.projHost != nil {
		if err := s.projHost.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("system: stop projection host: %w", err))
		}
	}

	for _, eng := range s.orderedEngines() {
		if err := eng.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Close external resources registered via RegisterCloser (after engines).
	for _, nc := range s.closers {
		if err := nc.closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("system: close %s: %w", nc.name, err))
		}
	}

	return errors.Join(errs...)
}

// GracefulClose drains in-flight work via any registered [Drainer] resources,
// then calls [Close] with a context-bounded timeout. It returns when Close
// finishes or the context is cancelled, whichever comes first. If draining
// fails — including context expiry — Close is still run unconditionally and
// the drain and close errors are joined: a spent drain deadline must not leak
// engines and registered resources. If the context expires during Close,
// resources may still be closing in the background.
//
// Use this instead of [Close] when you need a shutdown deadline (e.g., a
// Kubernetes SIGTERM grace period).
func (s *System) GracefulClose(ctx context.Context) error {
	// Phase 0: stop managed timers (scheduler dispatch must not race the
	// drain/close phases).
	s.stopTimers()

	// Phase 1: drain in-flight work. A drain error must not skip Close (see
	// godoc): the deadline spent draining cannot be allowed to leak every
	// engine and registered closer. Close runs unconditionally — not raced
	// against the (likely already expired) context — so the Phase 2
	// pre-expired-context determinism guard stays untouched.
	if drainErr := s.drainAll(ctx); drainErr != nil {
		return errors.Join(
			fmt.Errorf("system: graceful drain: %w", drainErr),
			s.Close(),
		)
	}

	// Phase 2: close with context race.
	done := make(chan error, 1)

	go func() { done <- s.Close() }()

	select {
	case err := <-done:
		// A pre-expired context must win deterministically over an
		// instantaneous Close: select's uniform choice between two ready
		// cases would otherwise flake (observed under parallel test load).
		if cerr := ctx.Err(); cerr != nil {
			return fmt.Errorf("system: graceful close: %w", cerr)
		}

		return err
	case <-ctx.Done():
		return fmt.Errorf("system: graceful close: %w", ctx.Err())
	}
}

// ResetProjection resets a projection's checkpoint and read-model state so it
// replays from the beginning on the next Start. The projection host must be
// stopped before calling this method (projectionhost.Host.Reset returns an
// error if the host is running).
//
// The projection must implement [projectionhost.Resettable] for its read-model
// state to be cleared. The checkpoint is always reset regardless.
func (s *System) ResetProjection(ctx context.Context, name string) error {
	if s.projHost == nil {
		return ErrNoProjectionHost
	}

	return s.projHost.Reset(ctx, name)
}
