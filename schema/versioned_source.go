package schema

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

var _ event.EventSource = (*VersionedStore)(nil)

type VersionedStore struct {
	inner event.Store

	registry *upcasterRegistry
}

func NewVersionedStore(store event.Store, upcasters ...Upcaster) (*VersionedStore, error) {
	if store == nil {
		return nil, event.NewRejection("schema.nil_store", "store is required")
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
	ref event.AggregateRef,
) ([]event.Event, error) {
	events, err := s.inner.Load(ctx, ref)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "schema.versioned_load",
			fmt.Sprintf("versioned store load %s", ref))
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) LoadFromVersion(
	ctx context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	events, err := s.inner.LoadFromVersion(ctx, ref, version)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "schema.versioned_load_from_version",
			fmt.Sprintf("versioned store load from version %s@%s", ref, version))
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) LoadToVersion(
	ctx context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	events, err := s.inner.LoadToVersion(ctx, ref, maxVersion)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "schema.versioned_load_to_version",
			fmt.Sprintf("versioned store load to version %s@%s", ref, maxVersion))
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) LoadToTimestamp(
	ctx context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	events, err := s.inner.LoadToTimestamp(ctx, ref, maxTime)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "schema.versioned_load_to_timestamp",
			fmt.Sprintf("versioned store load to timestamp %s@%s", ref,
				maxTime.Format(time.RFC3339)))
	}

	return s.upcastAll(events)
}

func (s *VersionedStore) Close() error {
	err := s.inner.Close()
	if err != nil {
		return event.WrapInfrastructure(err, "schema.versioned_close",
			"close versioned store")
	}

	return nil
}

func (s *VersionedStore) upcastAll(events []event.Event) ([]event.Event, error) {
	result := make([]event.Event, len(events))
	for i, evt := range events {
		upcasted, err := s.registry.upcast(evt)
		if err != nil {
			return nil, event.WrapCorruption(
				err,
				"schema.upcast_failed",
				"upcast event "+evt.ID().String(),
			)
		}

		result[i] = upcasted
	}

	return result, nil
}
