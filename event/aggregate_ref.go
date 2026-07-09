package event

import "github.com/larsartmann/go-cqrs-lite/id/v3"

// Deprecated: Use id.AggregateRef directly. This alias exists for backward
// compatibility and will be removed in v4.
type AggregateRef = id.AggregateRef

// Deprecated: Use id.NewAggregateRef directly. This alias exists for backward
// compatibility and will be removed in v4.
var NewAggregateRef = id.NewAggregateRef
