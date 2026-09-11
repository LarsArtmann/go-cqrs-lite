package stack

import (
	"context"
	"errors"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"
)

// shutdownEdge declares that `before` must be closed before `after` during
// Bundle.Close(). See [WithShutdownDependency].
type shutdownEdge struct {
	before, after io.Closer
}

// Close closes every resource registered with the Bundle, deduplicated by
// pointer so a *sql.DB shared across capabilities (via WithEventStore and
// WithCommandStore backed by the same SQLBackend) is closed exactly once.
//
// Returns the joined errors from every Close call. A nil return means every
// closer succeeded. Close is idempotent: subsequent calls are no-ops.
func (b *Bundle) Close() error {
	closers := b.orderedClosers()
	seen := make(map[io.Closer]struct{}, len(closers))

	var errs []error

	for _, c := range closers {
		if c == nil {
			continue
		}

		if _, dup := seen[c]; dup {
			continue
		}

		seen[c] = struct{}{}

		if err := c.Close(); err != nil { //nolint:noinlineerr // closer loop
			errs = append(errs, err)
		}
	}

	b.closers = nil

	return errors.Join(errs...)
}

// GracefulClose drains in-flight work, then closes the Bundle, all bounded by
// ctx. It runs two phases:
//
//  1. Drain: calls Drain on every registered [Drainer] (event subscribers,
//     projection runners, routers) so they stop accepting new work and finish
//     what's in flight. If any Drain returns an error, GracefulClose returns
//     it immediately without proceeding to Close.
//
//  2. Close: calls [Bundle.Close] to release all resources (stores, DB, bus).
//
// Both phases respect ctx: if the context is cancelled before a phase
// completes, the context error is returned. Resources may still be closing in
// the background — the caller should exit the process if the timeout fires.
//
// Use this instead of [Bundle.Close] when in-flight handlers may need time to
// drain. The closers list is ordered: the event bus is registered before
// stores, so it closes first, allowing BlockPublishUntilSubscriberAck to
// ensure ordered delivery completes.
func (b *Bundle) GracefulClose(ctx context.Context) error {
	for _, d := range b.drainers {
		if err := d.Drain(ctx); err != nil {
			return errorfamily.WrapInfrastructure(err, "stack.bundle.graceful_drain",
				"graceful drain")
		}
	}

	done := make(chan error, 1)

	go func() { done <- b.Close() }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("graceful close: %w", ctx.Err())
	}
}

// Drainer stops accepting new work and finishes in-flight work, bounded by
// ctx. Implemented by event subscribers, projection runners, and routers that
// need to drain before connections close. Resources that only need to release
// a handle (stores, DB connections) implement [io.Closer] instead.
//
// Drain is called by [Bundle.GracefulClose] BEFORE [Bundle.Close], so
// in-flight handlers complete before their underlying connections are dropped.
type Drainer interface {
	Drain(ctx context.Context) error
}

// registerCloser adds c to the list of resources [Bundle.Close] will release,
// if c implements [io.Closer]. Called by options that hand the Bundle a store,
// bus, or backend. Core interfaces no longer embed io.Closer (ADR-0010), so the
// type assertion is intentional: only resources that actually own a Close land
// in the list. Safe to call with the same closer multiple times — Close
// deduplicates by pointer.
func (b *Bundle) registerCloser(c any) {
	if c == nil {
		return
	}

	cl, ok := c.(io.Closer)
	if !ok {
		return
	}

	b.closers = append(b.closers, cl)
}
