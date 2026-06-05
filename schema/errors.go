package schema

import "github.com/larsartmann/go-cqrs-lite/event/v2"

var (
	// ErrNilStore is returned when a nil event.Store is passed to NewVersionedStore.
	ErrNilStore = event.NewRejection("schema.nil_store", "store is required")

	// ErrNilUpcaster is returned when an upcaster with a nil function is called.
	ErrNilUpcaster = event.NewRejection("schema.nil_upcaster", "upcaster function is nil")
)
