package otel

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

// Semantic attribute keys for CQRS telemetry.
// Follows OpenTelemetry messaging semantic conventions where applicable.
const (
	// MessageKind identifies the CQRS message kind: "command", "event", or "query".
	AttrMessageKind = "cqrs.message.kind"

	// CommandType is the command type identifier.
	AttrCommandType = "cqrs.command.type"

	// EventType is the event type identifier.
	AttrEventType = "cqrs.event.type"

	// QueryType is the query type identifier.
	AttrQueryType = "cqrs.query.type"

	// AggregateType is the aggregate root type.
	AttrAggregateType = "cqrs.aggregate.type"

	// AggregateID is the aggregate instance identifier.
	AttrAggregateID = "cqrs.aggregate.id"

	// AggregateVersion is the aggregate stream version.
	AttrAggregateVersion = "cqrs.aggregate.version"

	// EventCount is the number of events in a batch.
	AttrEventCount = "cqrs.event.count"

	// ProjectionName is the name of a projection.
	AttrProjectionName = "cqrs.projection.name"

	// SagaType is the saga definition type.
	AttrSagaType = "cqrs.saga.type"

	// SagaStep is the saga step index.
	AttrSagaStep = "cqrs.saga.step"

	// SagaStepName is the human-readable name of the saga step.
	AttrSagaStepName = "cqrs.saga.step_name"

	// OutboxEntryCount is the number of entries in an outbox operation.
	AttrOutboxEntryCount = "cqrs.outbox.entry_count"

	// Status indicates operation result: "success" or "error".
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

// AggregateAttrs returns the standard set of aggregate attributes for a span.
func AggregateAttrs(aggregateType string, aggregateID fmt.Stringer, version int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrAggregateType, aggregateType),
		attribute.String(AttrAggregateID, aggregateID.String()),
		attribute.Int(AttrAggregateVersion, version),
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
func EventAttrs(eventType string, aggregateID fmt.Stringer, aggregateType string) []attribute.KeyValue {
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
