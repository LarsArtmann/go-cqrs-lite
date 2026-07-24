package otel

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

// Semantic attribute keys for CQRS telemetry.
// Follows OpenTelemetry messaging semantic conventions where applicable.
const (
	// AttrMessageKind identifies the CQRS message kind: "command", "event", or "query".
	AttrMessageKind = "cqrs.message.kind"

	// AttrCommandType is the command type identifier.
	AttrCommandType = "cqrs.command.type"

	// AttrEventType is the event type identifier.
	AttrEventType = "cqrs.event.type"

	// AttrQueryType is the query type identifier.
	AttrQueryType = "cqrs.query.type"

	// The following AttrStream* constants use cqrs.aggregate.* string values
	// intentionally. Per ADR-0058, OTel attribute keys are operational schema
	// for observability pipelines (Grafana, Datadog, Prometheus dashboards),
	// the same stability category as JSON struct tags, slog field keys, and
	// error classification codes. Renaming the wire values would break every
	// consumer's dashboard/alert filters. The Go const NAMES are already
	// Stream* (the code-facing API); only the emitted key strings are frozen.

	// AttrStreamType is the stream type.
	AttrStreamType = "cqrs.aggregate.type"

	// AttrStreamID is the stream instance identifier.
	AttrStreamID = "cqrs.aggregate.id"

	// AttrStreamVersion is the stream version.
	AttrStreamVersion = "cqrs.aggregate.version"

	// AttrEventCount is the number of events in a batch.
	AttrEventCount = "cqrs.event.count"

	// AttrStreamCount is the number of streams in a multi-stream batch.
	AttrStreamCount = "cqrs.aggregate.count"

	// AttrProjectionName is the name of a projection.
	AttrProjectionName = "cqrs.projection.name"

	// AttrStatus indicates operation result: "success" or "error".
	AttrStatus = "cqrs.status"

	// StatusSuccess is the value for successful operations.
	StatusSuccess = "success"

	// StatusError is the value for failed operations.
	StatusError = "error"
)

// MessageKind values.
const (
	KindCommand = "command"
	KindEvent   = "event"
	KindQuery   = "query"
)

// StreamAttrs returns the stream type and ID attributes for a span.
func StreamAttrs(streamType, streamID fmt.Stringer) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrStreamType, streamType.String()),
		attribute.String(AttrStreamID, streamID.String()),
	}
}

// Deprecated: use AttrStreamType.
const AttrAggregateType = AttrStreamType

// Deprecated: use AttrStreamID.
const AttrAggregateID = AttrStreamID

// Deprecated: use AttrStreamVersion.
const AttrAggregateVersion = AttrStreamVersion

// Deprecated: use AttrStreamCount.
const AttrAggregateCount = AttrStreamCount

// Deprecated: use StreamAttrs.
func AggregateAttrs(streamType, streamID fmt.Stringer) []attribute.KeyValue {
	return StreamAttrs(streamType, streamID)
}

// CommandAttrs returns the standard set of command attributes for a span.
func CommandAttrs(commandType string, streamID fmt.Stringer) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindCommand),
		attribute.String(AttrCommandType, commandType),
		attribute.String(AttrStreamID, streamID.String()),
	}
}

// EventAttrs returns the standard set of event attributes for a span.
func EventAttrs(
	eventType string,
	streamID fmt.Stringer,
	streamType string,
) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindEvent),
		attribute.String(AttrEventType, eventType),
		attribute.String(AttrStreamID, streamID.String()),
		attribute.String(AttrStreamType, streamType),
	}
}

// QueryAttrs returns the standard set of query attributes for a span.
func QueryAttrs(queryType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindQuery),
		attribute.String(AttrQueryType, queryType),
	}
}
