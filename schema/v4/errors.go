package schema

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	// ErrNilStore is returned when a nil event.Store is passed to NewVersionedStore.
	ErrNilStore error = errorfamily.NewRejection("schema.nil_store", "store is required")

	// ErrNilJournal is returned when a nil event.SeekableJournal is passed to NewVersionedSeekableJournal.
	ErrNilJournal error = errorfamily.NewRejection("schema.nil_journal", "journal is required")

	// ErrNilUpcaster is returned when an upcaster with a nil function is called.
	ErrNilUpcaster error = errorfamily.NewRejection(
		"schema.nil_upcaster",
		"upcaster function is nil",
	)

	// ErrInvalidUpcastResult is returned when an upcaster returns nil or the
	// input event itself. Upcasters must return a NEW ImmutableEvent
	// instance: events are shared, and mutating the input corrupts the store.
	ErrInvalidUpcastResult error = errorfamily.NewCorruption(
		"schema.invalid_upcast_result",
		"upcaster must return a new event instance, not nil or the input",
	)

	// ErrInvalidOp is returned by Compile when an op declares invalid
	// parameters (empty event type, non-positive schema version, nil
	// function, split without outputs).
	ErrInvalidOp error = errorfamily.NewRejection(
		"schema.invalid_op",
		"op declares invalid parameters",
	)

	// ErrDuplicateOp is returned by Compile when two ops claim the same
	// match: the same (event type, schema version), or two type-only ops
	// (RenameType/Drop) on one event type. Declaration ambiguity is a
	// configuration error, never resolved silently.
	ErrDuplicateOp error = errorfamily.NewRejection(
		"schema.duplicate_op",
		"two ops match the same event type and schema version",
	)

	// ErrRenameCycle is returned by Compile when RenameType ops form a
	// cycle (directly or through chained renames) or a rename target is
	// claimed twice.
	ErrRenameCycle error = errorfamily.NewRejection(
		"schema.rename_cycle",
		"RenameType ops form a cycle or ambiguous identity",
	)

	// ErrBatchOpNotConvertible is returned by Chain.Upcasters when the chain
	// contains Split, Drop, or RenameType ops: they change event identity or
	// count and need the batch-level Chain.SourceTransform().
	ErrBatchOpNotConvertible error = errorfamily.NewRejection(
		"schema.op_not_convertible",
		"op needs batch semantics, not the single-event Upcaster interface",
	)

	// ErrChainCycle is returned when a compiled chain re-matches an event
	// more times than it has ops — a chain that cannot terminate. Compile
	// rejects the known static cycles; this guards the runtime remainder.
	ErrChainCycle error = errorfamily.NewCorruption(
		"schema.chain_cycle",
		"upcast chain does not terminate",
	)
)
