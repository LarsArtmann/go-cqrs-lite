// Package command provides command dispatch with typed handlers, middleware chains,
// and lifecycle management for CQRS applications.
//
// Commands represent intents to change state. Each command is dispatched to a single
// registered handler, which validates business rules and produces events.
//
// # Quick Start
//
//	cmds := command.NewDispatcher()
//	cmds.Register("user.create", func(ctx context.Context, cmd command.Command) error {
//	    return handleCreate(cmd)
//	})
//	err := cmds.Dispatch(ctx, cmd)
//
// # Typed Handlers
//
// For type safety, use RegisterTyped to avoid manual type assertions:
//
//	command.RegisterTyped[CreateUserCmd](cmds, "user.create",
//	    func(ctx context.Context, cmd *CreateUserCmd) error {
//	        return handleCreate(cmd)
//	    },
//	)
//
// # Middleware
//
// Middleware wraps handlers in a chain: the FIRST middleware added is the
// outermost wrapper, so it runs first and its post-processing runs last
// (dispatcher.Apply wraps in reverse registration order). Order matters for
// retry: a retry middleware placed OUTSIDE idempotency re-enters the dedup
// check on every attempt and fails the second attempt with ErrDuplicate.
//
//	cmds.Use(middleware.CommandLogging(logger))   // outermost: sees the final error
//	cmds.Use(middleware.CommandRecovery())
//	cmds.Use(middleware.CommandIdempotency(store, ttl, nil)) // once, before retries
//	cmds.Use(middleware.CommandRetry(cfg))
//
// # Command Persistence (Audit Trail)
//
// PersistedCommand captures every received command with full audit metadata
// (type, stream ref, payload, received-at timestamp). Use a CommandStore
// to save and load commands for audit and replay:
//
//	store := memory.NewMemoryCommandStore()
//	cmd, _ := command.NewPersistedCommand("user.create", ref, payload)
//	store.Save(ctx, ref, cmd)
//	loaded, _ := store.Load(ctx, ref)
//
// For cross-stream audit, use the CommandJournal interface:
//
//	all, _ := store.ReadAll(ctx)              // all commands, ordered by received_at
//	page, _ := store.ReadFrom(ctx, lastID, 100) // position-based pagination
package command
