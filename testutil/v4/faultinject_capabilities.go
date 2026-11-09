//nolint:wrapcheck // deliberate pass-through: this wrapper must not rewrite the inner error
package testutil

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// SaveMultiBatch forwards to the inner MultiSink, or fails loudly when the
// wrapped store has none.
//
// Capability forwarding lives in its own file so the base store wrapper stays
// readable: every optional interface (ADR-0126) is a deliberate addition here,
// and an interface missing from this file is a capability the injector silently
// drops.
func (f *FaultInjector) SaveMultiBatch(ctx context.Context, entries []event.MultiBatchEntry) error {
	if err := f.arm(MethodSaveMultiBatch); err != nil {
		return err
	}

	multi, ok := f.inner.(event.MultiSink)
	if !ok {
		return unavailable(MethodSaveMultiBatch)
	}

	return multi.SaveMultiBatch(ctx, entries)
}

// ReadAll forwards to the inner Journal.
func (f *FaultInjector) ReadAll(ctx context.Context) ([]event.Event, error) {
	if err := f.arm(MethodReadAll); err != nil {
		return nil, err
	}

	journal, ok := f.inner.(event.Journal)
	if !ok {
		return nil, unavailable(MethodReadAll)
	}

	return journal.ReadAll(ctx)
}

// ReadFrom forwards to the inner SeekableJournal.
func (f *FaultInjector) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	if err := f.arm(MethodReadFrom); err != nil {
		return nil, err
	}

	seekable, ok := f.inner.(event.SeekableJournal)
	if !ok {
		return nil, unavailable(MethodReadFrom)
	}

	return seekable.ReadFrom(ctx, afterEventID, limit)
}

// ReadStream forwards to the inner StreamingJournal.
func (f *FaultInjector) ReadStream( //nolint:ireturn // mirrors event.StreamingJournal, which returns the interface by contract
	ctx context.Context,
) (event.EventIterator, error) {
	if err := f.arm(MethodReadStream); err != nil {
		return nil, err
	}

	streaming, ok := f.inner.(event.StreamingJournal)
	if !ok {
		return nil, unavailable(MethodReadStream)
	}

	return streaming.ReadStream(ctx)
}

// ReadStreamFrom forwards to the inner StreamingJournal.
func (f *FaultInjector) ReadStreamFrom( //nolint:ireturn // mirrors event.StreamingJournal, which returns the interface by contract
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (event.EventIterator, error) {
	if err := f.arm(MethodReadStreamFrom); err != nil {
		return nil, err
	}

	streaming, ok := f.inner.(event.StreamingJournal)
	if !ok {
		return nil, unavailable(MethodReadStreamFrom)
	}

	return streaming.ReadStreamFrom(
		ctx,
		afterEventID,
		limit,
	)
}

// LoadBackwards forwards to the inner BackwardsSource.
func (f *FaultInjector) LoadBackwards(
	ctx context.Context,
	ref id.StreamRef,
) ([]event.Event, error) {
	if err := f.arm(MethodLoadBackwards); err != nil {
		return nil, err
	}

	backwards, ok := f.inner.(event.BackwardsSource)
	if !ok {
		return nil, unavailable(MethodLoadBackwards)
	}

	return backwards.LoadBackwards(ctx, ref)
}

// Close forwards to the inner store when it is closable.
func (f *FaultInjector) Close() error {
	if err := f.arm(MethodClose); err != nil {
		return err
	}

	closer, ok := f.inner.(interface{ Close() error })
	if !ok {
		return nil
	}

	return closer.Close()
}
