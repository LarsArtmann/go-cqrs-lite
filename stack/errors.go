package stack

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Validation and accessor errors returned by [New] and the [Bundle] accessors.
//
// All classified as Rejection: a misconfigured Bundle is a non-retryable
// programmer error. Without this the retry middleware would treat them as
// Transient (the default for unclassified errors) and pointlessly retry.
var (
	// ErrEmpty is returned by [New] when no capability field is set on the
	// Bundle. An entirely empty Bundle is always a bug — at least one store,
	// bus, or backend must be configured.
	ErrEmpty = errorfamily.NewRejection(
		"stack.bundle_empty",
		"stack: bundle is empty (no capabilities set)",
	)

	// ErrMissingEventStore is returned by accessors that require a composite
	// event.Store. It means the Bundle was configured without
	// [WithEventStore] (or a preset that sets one).
	ErrMissingEventStore = errorfamily.NewRejection(
		"stack.missing_event_store",
		"stack: bundle has no event.Store (use WithEventStore or a preset)",
	)

	// ErrMissingPublisher is deprecated: stack.Repository no longer requires
	// a publisher. It is retained for consumers that reference it.
	ErrMissingPublisher = errorfamily.NewRejection(
		"stack.missing_publisher",
		"stack: bundle has no event.Publisher",
	)

	// ErrMissingReadModels is returned by [ReadModel] when the Bundle
	// has no read-model backend.
	ErrMissingReadModels = errorfamily.NewRejection(
		"stack.missing_read_models",
		"stack: bundle has no read-model backend (use WithReadModels)",
	)

	// ErrMissingJournal is returned by [Bundle.CatchUpSubscriber] when the
	// Bundle has no event.Journal for projection replay.
	ErrMissingJournal = errorfamily.NewRejection(
		"stack.missing_journal",
		"stack: bundle has no event.Journal",
	)

	// ErrMissingSubscriber is returned by [Bundle.CatchUpSubscriber] when the
	// Bundle has no event.Subscriber for live event handling.
	ErrMissingSubscriber = errorfamily.NewRejection(
		"stack.missing_subscriber",
		"stack: bundle has no event.Subscriber",
	)

	// ErrMissingCheckpoint is returned by [Bundle.CatchUpSubscriber] when the
	// Bundle has no checkpoint store for tracking projection position.
	ErrMissingCheckpoint = errorfamily.NewRejection(
		"stack.missing_checkpoint",
		"stack: bundle has no checkpoint store",
	)

	// ErrMissingQueryStore is returned by [QueryAuditMiddleware] when the
	// Bundle has no query sink for persisting audit records.
	ErrMissingQueryStore = errorfamily.NewRejection(
		"stack.missing_query_store",
		"stack: bundle has no query sink (use WithQueryStore)",
	)
)
