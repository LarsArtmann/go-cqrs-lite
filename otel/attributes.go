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

	// AttrAggregateType is the aggregate root type.
	AttrAggregateType = "cqrs.aggregate.type"

	// AttrAggregateID is the aggregate instance identifier.
	AttrAggregateID = "cqrs.aggregate.id"

	// AttrAggregateVersion is the aggregate stream version.
	AttrAggregateVersion = "cqrs.aggregate.version"

	// AttrEventCount is the number of events in a batch.
	AttrEventCount = "cqrs.event.count"

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

// AggregateAttrs returns the aggregate type and ID attributes for a span.
func AggregateAttrs(aggregateType, aggregateID fmt.Stringer) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrAggregateType, aggregateType.String()),
		attribute.String(AttrAggregateID, aggregateID.String()),
	}
}

// CommandAttrs returns the standard set of command attributes for a span.
func CommandAttrs(commandType string, aggregateID fmt.Stringer) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindCommand),
		attribute.String(AttrCommandType, commandType),
		attribute.String(AttrAggregateID, aggregateID.String()),
	}
}

// EventAttrs returns the standard set of event attributes for a span.
func EventAttrs(
	eventType string,
	aggregateID fmt.Stringer,
	aggregateType string,
) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindEvent),
		attribute.String(AttrEventType, eventType),
		attribute.String(AttrAggregateID, aggregateID.String()),
		attribute.String(AttrAggregateType, aggregateType),
	}
}

// QueryAttrs returns the standard set of query attributes for a span.
func QueryAttrs(queryType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindQuery),
		attribute.String(AttrQueryType, queryType),
	}
}
