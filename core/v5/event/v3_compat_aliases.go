package event

import (
	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/metadata"
)

// Backward-compatibility aliases for consumers written against the v3.7.x API
// where these types lived in the event package. They were moved to id/metadata
// but downstream modules (e.g., cqrs-htmx/usermgmt) still reference them here.

// art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
type StreamType = id.StreamType

// Deprecated: use id.StreamType. Retained for v3 consumers that reference event.AggregateType.
type AggregateType = id.StreamType

// Deprecated: use id.StreamID. Retained for v3 consumers that reference event.AggregateID.
type AggregateID = id.StreamID

// Deprecated: use id.StreamRef. Retained for v3 consumers that reference event.AggregateRef.
type AggregateRef = id.StreamRef

// Deprecated: use id.ParseStreamType.
var ParseAggregateType = id.ParseAggregateType

type StreamRef = id.StreamRef

// Deprecated: use id.NewStreamRef.
var NewAggregateRef = id.NewStreamRef

type CustomData[K ~string] = metadata.CustomData[K]
