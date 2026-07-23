package event

import (
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

// Backward-compatibility aliases for consumers written against the v3.7.x API
// where these types lived in the event package. They were moved to id/metadata
// but downstream modules (e.g., cqrs-htmx/usermgmt) still reference them here.

type StreamType = id.StreamType

var ParseAggregateType = id.ParseAggregateType

type StreamRef = id.StreamRef

var NewAggregateRef = id.NewStreamRef

type CustomData[K ~string] = metadata.CustomData[K]
