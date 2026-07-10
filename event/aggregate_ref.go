package event

import "github.com/larsartmann/go-cqrs-lite/id/v3"

// Deprecated: Use id.AggregateRef directly. This alias exists for backward
// compatibility and will be removed in v4.
// v4-removal: remove this alias and update all consumers to import id/ directly.
type AggregateRef = id.AggregateRef

// Deprecated: Use id.NewAggregateRef directly. This alias exists for backward
// compatibility and will be removed in v4.
// v4-removal: remove this alias.
var NewAggregateRef = id.NewAggregateRef
