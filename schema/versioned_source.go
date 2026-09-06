package schema

import (
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// UpcastSourceTransform returns an [event.SourceTransform] that applies the
// given upcasters to every loaded event. Compose it with
// [event.DecorateStore] to add schema evolution to any event store:
//
//	store := event.DecorateStore(raw, nil, schema.UpcastSourceTransform(u1, u2))
//
// Unlike the deprecated VersionedStore, the decorated store preserves ALL
// inner-store capabilities: Journal, SeekableJournal, BackwardsSource,
// MultiSink, and io.Closer all keep working with upcasting applied.
func UpcastSourceTransform(upcasters ...Upcaster) event.SourceTransform {
	registry := newUpcasterRegistryFrom(upcasters)

	return registry.upcastAll
}

// VersionedStore is the pre-transform wrapper around an event store with
// upcaster support.
//
// Deprecated: Use [UpcastSourceTransform] with [event.DecorateStore]:
//
//	versioned := event.DecorateStore(store, nil, schema.UpcastSourceTransform(uc))
//
// The deprecated shell still works: it embeds the decorated store, so all
// Store methods are available, and it additionally forwards Close.
// Removed at v5 (ADR-0126).
type VersionedStore struct {
	event.Store
}

// NewVersionedStore wraps an event.Store with upcaster support on reads.
//
// Deprecated: Use [UpcastSourceTransform] with [event.DecorateStore]. Kept
// so existing consumers keep compiling; returns the compatibility shell.
// Removed at v5 (ADR-0126).
func NewVersionedStore(store event.Store, upcasters ...Upcaster) (*VersionedStore, error) {
	if store == nil {
		return nil, ErrNilStore
	}

	return &VersionedStore{
		Store: event.DecorateStore(store, nil, UpcastSourceTransform(upcasters...)),
	}, nil
}

// Close closes the underlying store when it implements io.Closer.
func (s *VersionedStore) Close() error {
	c, ok := s.Store.(io.Closer)
	if !ok {
		return nil
	}

	if err := c.Close(); err != nil {
		return errorfamily.WrapInfrastructure(err, "schema.versioned_close",
			"close versioned store")
	}

	return nil
}
