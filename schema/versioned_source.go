package schema

import (
	"context"
	"fmt"
	"io"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

var _ event.EventSource = (*VersionedStore)(nil)

type VersionedStore struct {
	inner event.Store

	registry *upcasterRegistry
}

func NewVersionedStore(store event.Store, upcasters ...Upcaster) (*VersionedStore, error) {
	if store == nil {
		return nil, ErrNilStore
	}

	reg := newUpcasterRegistry()

	for _, u := range upcasters {
		if u != nil {
			reg.register(u)
		}
	}

	return &VersionedStore{inner: store, registry: reg}, nil
}

func (s *VersionedStore) Load(
	ctx context.Context,
	ref id.AggregateRef,
) ([]event.Event, error) {
	return s.loadAndUpcast(func() ([]event.Event, error) {
		return s.inner.Load(ctx, ref)
	}, "schema.versioned_load", fmt.Sprintf("versioned store load %s", ref))
}

func (s *VersionedStore) LoadFromVersion(
	ctx context.Context,
	ref id.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	return s.loadAndUpcast(func() ([]event.Event, error) {
		return s.inner.LoadFromVersion(ctx, ref, version)
	}, "schema.versioned_load_from_version",
		fmt.Sprintf("versioned store load from version %s@%s", ref, version))
}

func (s *VersionedStore) LoadToVersion(
	ctx context.Context,
	ref id.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	return s.loadAndUpcast(func() ([]event.Event, error) {
		return s.inner.LoadToVersion(ctx, ref, maxVersion)
	}, "schema.versioned_load_to_version",
		fmt.Sprintf("versioned store load to version %s@%s", ref, maxVersion))
}

func (s *VersionedStore) LoadToTimestamp(
	ctx context.Context,
	ref id.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	return s.loadAndUpcast(func() ([]event.Event, error) {
		return s.inner.LoadToTimestamp(ctx, ref, maxTime)
	}, "schema.versioned_load_to_timestamp",
		fmt.Sprintf("versioned store load to timestamp %s@%s", ref,
			maxTime.Format(time.RFC3339)))
}

func (s *VersionedStore) loadAndUpcast(
	load func() ([]event.Event, error),
	code, msg string,
) ([]event.Event, error) {
	events, err := load()
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, code, msg)
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) Close() error {
	if c, ok := s.inner.(io.Closer); ok {
		err := c.Close()
		if err != nil {
			return errorfamily.WrapInfrastructure(err, "schema.versioned_close",
				"close versioned store")
		}
	}

	return nil
}

func (s *VersionedStore) upcastAll(events []event.Event) ([]event.Event, error) {
	result := make([]event.Event, len(events))
	for i, evt := range events {
		upcasted, err := s.registry.upcast(evt)
		if err != nil {
			return nil, errorfamily.WrapCorruption(
				err,
				"schema.upcast_failed",
				"upcast event "+evt.ID().String(),
			)
		}

		result[i] = upcasted
	}

	return result, nil
}
