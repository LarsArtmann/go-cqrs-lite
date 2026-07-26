package metaengine

import (
	"encoding/json/v2"
	"fmt"
	"reflect"
)

// reify converts a loosely-typed value (typically map[string]any from a SQL
// engine's JSON decode) into the target type R via JSON round-trip. Returns an
// error if either marshal or unmarshal fails; callers fall through to the
// type-mismatch error if reification is impossible.
//
// This bridges the cross-engine divergence: memory engines store and return
// typed Go values directly, while SQL engines JSON-encode on write and decode
// to any on read — producing map[string]any for structs. reify rebuilds the
// typed value so ExecuteTyped[R] works identically regardless of engine.
func reify[R any](raw any) (R, error) {
	var zero R

	b, err := json.Marshal(raw)
	if err != nil {
		return zero, fmt.Errorf("metaengine.reify: marshal value: %w", err)
	}

	var r R

	if err := json.Unmarshal(b, &r); err != nil {
		return zero, fmt.Errorf("metaengine.reify: unmarshal into %T: %w", r, err)
	}

	return r, nil
}

// reifyReflect converts value into a reflect.Value assignable to target.
//
// Memory engines store and return typed Go values directly, so value is
// already assignable and is returned as-is (no JSON round-trip, no alloc).
// SQL engines JSON-encode on write and decode into any on read, producing
// map[string]any for structs — which is not assignable to a typed parameter
// and would panic inside a reflect.Call. reifyReflect rebuilds the typed
// value via JSON round-trip, mirroring reify[R] above for the reflect call
// sites that do not have a static type parameter.
//
// Reification cannot fail for values an engine itself wrote (they are valid
// JSON of exactly target), so the round-trip is lossless. A marshal/unmarshal
// failure (only possible for externally-corrupted data, or for a raw cursor
// scalar passed where a struct is expected) falls back to the zero value of
// target rather than panicking.
func reifyReflect(value any, target reflect.Type) reflect.Value {
	if rt := reflect.TypeOf(value); rt != nil && rt.AssignableTo(target) {
		return reflect.ValueOf(value)
	}

	b, err := json.Marshal(value)
	if err != nil {
		return reflect.Zero(target)
	}

	v := reflect.New(target)

	if err := json.Unmarshal(b, v.Interface()); err != nil {
		return reflect.Zero(target)
	}

	return v.Elem()
}
