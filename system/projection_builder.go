package system

import (
	"fmt"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// LookupInput is the default query input type for auto-generated projections.
// It carries just the key for point lookups. For filtered/sorted queries,
// define a custom input struct and use explicit metaengine.Query.
type LookupInput[K any] struct {
	ID K
}

// ProjectionSpec is a type-erased projection declaration created by [View] or
// [Count]. It carries a build closure that generates folds and decoder entries
// when system.New() processes it.
//
// Consumers create ProjectionSpec values via [View] and pass them in
// DomainConfig.Projections. The system constructor detects ProjectionSpec
// values (vs raw metaengine.QueryDecl values) and processes them automatically.
type ProjectionSpec struct {
	name string
	// build generates the query declaration (as any, since QueryDecl is generic),
	// decoder entries for TypeDecoder registration, and any error.
	build func() (queryDecl any, decoderEntries []decoderEntry, err error)
}

// decoderEntry pairs a wire event type with its sample struct for TypeDecoder
// registration.
type decoderEntry struct {
	eventType string
	sample    any
}

// viewBuilder is the fluent builder for CRUD-shaped view projections.
// Created by [View], finalized by [.From].
type viewBuilder[R any, K any] struct {
	name     string
	keyField string
}

// View declares a CRUD-shaped view projection that auto-generates insert,
// update, and delete folds from event struct field matching (ADR-0116 Layer 1).
//
// The type parameters are:
//   - R: the result/view type (e.g. UserView)
//   - K: the key type (e.g. UserID)
//
// The key field defaults to "ID". Override with [.Key].
//
// Example:
//
//	system.View[UserView, UserID]("users").
//	    From(
//	        system.Event("user.created", UserCreated{}),
//	        system.Event("user.updated", UserUpdated{}),
//	        system.Event("user.deleted", UserDeleted{}),
//	    )
func View[R any, K any](name string) *viewBuilder[R, K] {
	return &viewBuilder[R, K]{name: name}
}

// Key overrides the key field name (default: "ID").
func (b *viewBuilder[R, K]) Key(field string) *viewBuilder[R, K] {
	b.keyField = field

	return b
}

// From finalizes the projection spec by providing the event samples.
// Each sample pairs a wire event type string with a sample struct whose name
// ends in Created/Updated/Deleted (ADR-0116 convention).
//
// Use [Event] (an alias for [metaengine.NamedEvent]) to create samples:
//
//	system.View[UserView, UserID]("users").From(
//	    system.Event("user.created", UserCreated{}),
//	    system.Event("user.deleted", UserDeleted{}),
//	)
func (b *viewBuilder[R, K]) From(samples ...metaengine.NamedSample) ProjectionSpec {
	keyField := b.keyField
	if keyField == "" {
		keyField = "ID"
	}

	samplesCopy := append([]metaengine.NamedSample(nil), samples...)

	return ProjectionSpec{
		name: b.name,
		build: func() (any, []decoderEntry, error) {
			folds, err := metaengine.AutoCRUDByNamedEvents[R](keyField, samplesCopy...)
			if err != nil {
				return nil, nil, fmt.Errorf("system: projection %q: %w", b.name, err)
			}

			foldArgs := make([]any, len(folds))
			for i, f := range folds {
				foldArgs[i] = f
			}

			query := metaengine.Query[LookupInput[K], R](b.name, foldArgs...)

			entries := make([]decoderEntry, len(samplesCopy))
			for i, s := range samplesCopy {
				entries[i] = decoderEntry{
					eventType: s.EventType(),
					sample:    s.Sample(),
				}
			}

			return query, entries, nil
		},
	}
}

// Event is an alias for [metaengine.NamedEvent], exported here for ergonomic
// one-package import when declaring projections:
//
//	system.View[UserView, UserID]("users").From(
//	    system.Event("user.created", UserCreated{}),
//	)
func Event(eventType string, sample any) metaengine.NamedSample {
	return metaengine.NamedEvent(eventType, sample)
}

// buildProjection processes ProjectionSpec values from DomainConfig.Projections,
// separating them from raw QueryDecl values. It calls each spec's build closure
// to generate folds and decoder entries, then returns:
//   - queryDecls: the generated QueryDecl values (as []any for Plan)
//   - typeDecoder: a TypeDecoder with all auto-generated entries (may be nil)
func buildProjection(projections []any) (queryDecls []any, typeDecoder *projectionadapter.TypeDecoder, err error) {
	var allEntries []decoderEntry

	for _, proj := range projections {
		spec, ok := proj.(ProjectionSpec)
		if !ok {
			// Raw QueryDecl — pass through unchanged.
			queryDecls = append(queryDecls, proj)

			continue
		}

		queryDecl, entries, buildErr := spec.build()
		if buildErr != nil {
			return nil, nil, buildErr
		}

		queryDecls = append(queryDecls, queryDecl)
		allEntries = append(allEntries, entries...)
	}

	if len(allEntries) > 0 {
		regs := make([]projectionadapter.EventRegistration, len(allEntries))

		for i, e := range allEntries {
			regs[i] = buildRegistration(e)
		}

		typeDecoder = projectionadapter.NewTypeDecoder(regs...)
	}

	return queryDecls, typeDecoder, nil
}

// buildRegistration creates a projectionadapter.EventRegistration from a decoder
// entry using reflection to avoid requiring a generic function per event type.
func buildRegistration(e decoderEntry) projectionadapter.EventRegistration {
	sampleType := reflect.TypeOf(e.sample)

	return projectionadapter.RegisterString(e.eventType, e.sample).
		WithDecoder(func(payload []byte) (any, error) {
			val := reflect.New(sampleType).Interface()

			if err := decodePayload(payload, val); err != nil {
				return nil, err
			}

			return reflect.ValueOf(val).Elem().Interface(), nil
		})
}

// decodePayload unmarshals a JSON payload into the target value.
// It uses encoding/json/v2 via the event module's codec.
func decodePayload(payload []byte, target any) error {
	if len(payload) == 0 {
		return nil
	}

	return jsonUnmarshal(payload, target)
}
