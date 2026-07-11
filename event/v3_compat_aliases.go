package event

import (
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/metadata/v3"
)

// Backward-compatibility aliases for consumers written against the v3.7.x API
// where these types lived in the event package. They were moved to id/metadata
// but downstream modules (e.g., cqrs-htmx/usermgmt) still reference them here.

type AggregateType = id.AggregateType

var ParseAggregateType = id.ParseAggregateType

type AggregateRef = id.AggregateRef

var NewAggregateRef = id.NewAggregateRef

type CustomData[K ~string] = metadata.CustomData[K]
