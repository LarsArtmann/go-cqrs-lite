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
	ErrNilUpcaster error = errorfamily.NewRejection("schema.nil_upcaster", "upcaster function is nil")

	// ErrInvalidUpcastResult is returned when an upcaster returns nil or the
	// input event itself. Upcasters must return a NEW ImmutableEvent
	// instance: events are shared, and mutating the input corrupts the store.
	ErrInvalidUpcastResult error = errorfamily.NewCorruption(
		"schema.invalid_upcast_result",
		"upcaster must return a new event instance, not nil or the input",
	)
)
