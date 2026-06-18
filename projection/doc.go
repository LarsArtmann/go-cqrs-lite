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
// Use NewBuilder to construct a projection, then register typed handlers
// with the generic On function:
//
//	builder := projection.NewBuilder("user-projection")
//
//	err := projection.On[UserCreated](builder, "user.created", codec.JSONCodec{},
//	    func(ctx context.Context, payload UserCreated) error {
//	        return updateUserReadModel(payload)
//	    },
//	)
//
//	err = projection.On[UserDeleted](builder, "user.deleted", codec.JSONCodec{},
//	    func(ctx context.Context, payload UserDeleted) error {
//	        return removeUserReadModel(payload)
//	    },
//	)
//
//	proj := builder.Build()
//	runner, _ := projection.NewRunner(store, bus, checkpointStore)
//	_ = runner.Register(proj)
//	go runner.Run(ctx)
//
// # Read-Your-Writes: RunReplay + RunLive
//
// Run is a convenience wrapper around two phases. For read-your-writes
// consistency (e.g. in tests or right after startup), call them separately so
// the read model is guaranteed caught up before you serve traffic:
//
//	if err := runner.RunReplay(ctx); err != nil { return err } // synchronous
//	go func() { _ = runner.RunLive(ctx) }()                    // background tail
//	// read model is caught up here — no time.Sleep needed
//
// # Handler Registration
//
// On[T] is a package-level generic function (not a method on Builder) that
// decodes event payloads using the provided codec before passing them to
// the handler. For raw event access, use HandlerRegistry.On directly.
package projection
