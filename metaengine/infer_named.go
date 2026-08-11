package metaengine

import (
	"fmt"
	"reflect"
)

// namedInferenceRequest is a marker type that tells Query() to defer fold
// generation to Plan() time using NamedSample values that pair wire event
// type strings with Go struct samples. This is the production counterpart to
// [Infer] for event pipelines that use dot-separated event types.
type namedInferenceRequest struct {
	samples []NamedSample
}

// InferFromNamedEvents requests planner-time fold inference from NamedSample
// values. Each NamedSample pairs a wire event type string ("user.created")
// with a Go struct sample (UserCreated{}) whose name must end in
// Created/Updated/Deleted.
//
// This is the production counterpart to [Infer] for consumers using the event
// pipeline with dot-separated event types. The planner classifies samples by
// Go struct name suffix, generates folds via field-name matching, then
// overrides the fold event types with the wire types.
//
// Same disclaimer as Infer applies: prefer explicit folds for production
// domain models.
//
// Example:
//
//	q := metaengine.Query[GetUser, UserView]("users",
//	    metaengine.InferFromNamedEvents(
//	        metaengine.NamedEvent("user.created", UserCreated{}),
//	        metaengine.NamedEvent("user.deleted", UserDeleted{}),
//	    ),
//	)
func InferFromNamedEvents(samples ...NamedSample) namedInferenceRequest {
	if len(samples) == 0 {
		panic("metaengine.InferFromNamedEvents: at least one named sample required")
	}

	for i, s := range samples {
		if s.eventType == "" {
			panic(fmt.Sprintf(
				"metaengine.InferFromNamedEvents: sample[%d] has empty event type", i,
			))
		}

		t := reflect.TypeOf(s.sample)
		if t == nil {
			panic(fmt.Sprintf(
				"metaengine.InferFromNamedEvents: sample[%d] is a nil interface", i,
			))
		}

		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		if t.Kind() != reflect.Struct {
			panic(fmt.Sprintf(
				"metaengine.InferFromNamedEvents: sample[%d] (%s) must be a struct, got %s",
				i, t.Name(), t.Kind(),
			))
		}
	}

	return namedInferenceRequest{samples: samples}
}

// applyNamedEventTypes overrides the eventType field on generated folds with
// wire event type strings from the NamedSamples. The fold structs store
// eventType as the Go struct name (set by EventTypeName); this replaces it
// with the dot-separated wire type for event pipeline compatibility.
func applyNamedEventTypes(folds []Fold, samples []NamedSample) {
	nameToWire := make(map[string]string, len(samples))

	for _, s := range samples {
		t := reflect.TypeOf(s.sample)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		nameToWire[t.Name()] = s.eventType
	}

	for _, f := range folds {
		if wireType, ok := nameToWire[f.EventType()]; ok {
			overrideEventType(f, wireType)
		}
	}
}
