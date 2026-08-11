package system

import (
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
	// retry count, failure log, processing time) ready for DomainConfig.Projections.
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
