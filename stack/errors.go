package stack

import "errors"

// Validation and accessor errors returned by [New] and the [Bundle] accessors.
var (
	// ErrEmpty is returned by [New] when no capability field is set on the
	// Bundle. An entirely empty Bundle is always a bug — at least one store,
	// bus, or backend must be configured.
	ErrEmpty = errors.New("stack: bundle is empty (no capabilities set)")

	// ErrMissingEventStore is returned by accessors that require a composite
	// event.Store. It means the Bundle was configured without
	// [WithEventStore] (or a preset that sets one).
	ErrMissingEventStore = errors.New(
		"stack: bundle has no event.Store (use WithEventStore or a preset)",
	)

	// ErrMissingPublisher is deprecated: stack.Repository no longer requires
	// a publisher. It is retained for consumers that reference it.
	ErrMissingPublisher = errors.New("stack: bundle has no event.Publisher")

	// ErrMissingReadModels is returned by [Bundle.ReadModel] when the Bundle
	// has no read-model backend.
	ErrMissingReadModels = errors.New(
		"stack: bundle has no read-model backend (use WithReadModels)",
	)

	// ErrMissingJournal is returned by [Bundle.ProjectionRunner] when the
	// Bundle has no event.Journal for projection replay.
	ErrMissingJournal = errors.New("stack: bundle has no event.Journal")

	// ErrMissingSubscriber is returned by [Bundle.ProjectionRunner] when the
	// Bundle has no event.Subscriber for live event handling.
	ErrMissingSubscriber = errors.New("stack: bundle has no event.Subscriber")

	// ErrMissingCheckpoint is returned by [Bundle.ProjectionRunner] when the
	// Bundle has no checkpoint store for tracking projection position.
	ErrMissingCheckpoint = errors.New("stack: bundle has no checkpoint store")
)
