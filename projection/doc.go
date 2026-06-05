// Package projection provides a replay+live projection runner for building
// read models from event streams.
//
// The Runner handles the full projection lifecycle:
//  1. Load checkpoint → replay events from that position
//  2. Switch to live subscription for new events
//  3. Retry failed events with configurable backoff
//  4. Route unrecoverable events to a dead letter handler
//
// # Quick Start
//
//	runner := projection.NewBuilder("user-projection").
//	    On("user.created", func(ctx context.Context, evt event.Event) error {
//	        return updateUserReadModel(evt)
//	    }).
//	    On("user.deleted", func(ctx context.Context, evt event.Event) error {
//	        return removeUserReadModel(evt)
//	    }).
//	    Runner(store, bus)
//
//	go runner.Run(ctx)
//
// # Builder Pattern
//
// Use projection.On[T] for type-safe event handling with automatic payload decoding.
// The Builder compiles event handlers into a projection.HandlerRegistry.
package projection
