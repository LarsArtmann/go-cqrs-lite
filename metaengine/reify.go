package metaengine

import "encoding/json/v2"

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
		return zero, err
	}

	var r R

	if err := json.Unmarshal(b, &r); err != nil {
		return zero, err
	}

	return r, nil
}
