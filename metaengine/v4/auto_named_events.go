package metaengine

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// NamedSample pairs a wire event type string with a sample struct for
// reflection-based fold generation. It bridges the gap between the
// dot-separated event types used in the event pipeline ("task.created")
// and the Go struct types used for field matching (TaskCreated).
//
// Created by [NamedEvent] and consumed by [AutoCRUDByNamedEvents].
type NamedSample struct {
	eventType string
	sample    any
}

// NamedEvent creates a NamedSample that pairs a wire event type string
// with a sample struct. The struct name must end in Created/Updated/Deleted
// to classify the fold kind (ADR-0116 Layer 1 convention).
//
//	folds, err := metaengine.AutoCRUDByNamedEvents[UserView]("ID",
//	    metaengine.NamedEvent("user.created", UserCreated{}),
//	    metaengine.NamedEvent("user.updated", UserUpdated{}),
//	    metaengine.NamedEvent("user.deleted", UserDeleted{}),
//	)
func NamedEvent(eventType string, sample any) NamedSample {
	return NamedSample{eventType: eventType, sample: sample}
}

// EventType returns the wire event type string for this sample.
func (s NamedSample) EventType() string { return s.eventType }

// Sample returns the sample struct value.
func (s NamedSample) Sample() any { return s.sample }

// AutoCRUDByNamedEvents generates insert, update, and delete folds by scanning
// sample struct names for Created/Updated/Deleted suffixes (ADR-0116 Layer 1),
// using the provided wire event type strings for fold matching instead of the
// Go struct names.
//
// This is the production variant of [AutoCRUDByConvention] for use with the
// event pipeline, where event types are dot-separated strings ("task.created")
// that differ from Go struct names ("TaskCreated").
//
// Returns an error if no Created sample is found (insert is the minimum
// requirement), if multiple samples match the same suffix, or if a sample is
// not a struct or has no exported field named keyField (such as an empty
// struct): the event type cannot be identified without its key.
func AutoCRUDByNamedEvents[R any](keyField string, samples ...NamedSample) ([]Fold, error) {
	resultType := reflect.TypeFor[R]()

	var created, updated, deleted *NamedSample
	var createdType, updatedType, deletedType reflect.Type

	for i := range samples {
		s := samples[i]

		t := reflect.TypeOf(s.sample)
		if t == nil {
			return nil, fmt.Errorf("AutoCRUDByNamedEvents: sample for event type %q is nil", s.eventType)
		}

		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		name := t.Name()

		switch {
		case strings.HasSuffix(name, "Created"):
			if created != nil {
				return nil, fmt.Errorf(
					"AutoCRUDByNamedEvents: multiple Created types: %s and %s",
					createdType.Name(), name,
				)
			}

			created = &samples[i]
			createdType = t
		case strings.HasSuffix(name, "Updated"):
			if updated != nil {
				return nil, fmt.Errorf(
					"AutoCRUDByNamedEvents: multiple Updated types: %s and %s",
					updatedType.Name(), name,
				)
			}

			updated = &samples[i]
			updatedType = t
		case strings.HasSuffix(name, "Deleted"):
			if deleted != nil {
				return nil, fmt.Errorf(
					"AutoCRUDByNamedEvents: multiple Deleted types: %s and %s",
					deletedType.Name(), name,
				)
			}

			deleted = &samples[i]
			deletedType = t
		default:
			return nil, fmt.Errorf(
				"AutoCRUDByNamedEvents: sample for event type %q has type %q, which does not match "+
					"the *Created/*Updated/*Deleted suffix convention",
				s.eventType, name,
			)
		}

		if err := namedSampleUsable(s, t, keyField); err != nil {
			return nil, err
		}
	}

	if created == nil {
		return nil, errors.New(
			"AutoCRUDByNamedEvents: no *Created sample provided (at least one is required)",
		)
	}

	var folds []Fold

	folds = append(folds, overrideEventType(
		autoInsertByType(createdType, resultType, keyField),
		created.eventType,
	))

	if updated != nil {
		folds = append(folds, overrideEventType(
			autoUpdateByType(updatedType, resultType, keyField),
			updated.eventType,
		))
	}

	if deleted != nil {
		folds = append(folds, overrideEventType(
			autoDeleteByType(deletedType, keyField),
			deleted.eventType,
		))
	}

	return folds, nil
}

// namedSampleUsable rejects a sample type the generated folds could not read
// the key from. The convention helpers panic on such input; this
// error-returning generator refuses it instead.
func namedSampleUsable(s NamedSample, t reflect.Type, keyField string) error {
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("AutoCRUDByNamedEvents: sample for event type %q is a %s, not a struct",
			s.eventType, t.Kind())
	}

	if _, err := findField(t, keyField); err != nil {
		return fmt.Errorf("AutoCRUDByNamedEvents: sample for event type %q (type %s) is unusable: %w",
			s.eventType, t.Name(), err)
	}

	return nil
}

// overrideEventType replaces the eventType field on a generated fold with the
// wire event type string. The fold structs (insertFold, updateFold, removeFold)
// store eventType as an unexported field set by EventTypeName(sample). This
// override is only callable from within the metaengine package.
func overrideEventType(fold Fold, wireType string) Fold {
	switch f := fold.(type) {
	case *insertFold:
		f.eventType = wireType
	case *updateFold:
		f.eventType = wireType
	case *removeFold:
		f.eventType = wireType
	}

	return fold
}
