// Package projections provides pre-built metaengine projection declarations
// for command lifecycle event streams (ADR-0117).
//
// These projections fold lifecycle events into query-optimized materialized
// views. The planner auto-routes them to the best available engine.
//
// | Projection        | Source events           | ADT     | Query                         |
// | ----------------- | ----------------------- | ------- | ----------------------------- |
// | Dead-letter queue | command.dead-lettered   | Map     | "Which commands are DL?"      |
// | Retry count       | command.retried         | Counter | "How many retries for cmd-X?" |
// | Failure log       | command.failed          | Log     | "Show recent failures"        |
//
// # Usage
//
//	store, _ := metaengine.Plan(engines,
//	    projections.DeadLetterQueue(),
//	    projections.RetryCount(),
//	    projections.FailureLog(),
//	)
//
//	// The projection host feeds lifecycle events to the store:
//	//   store.ApplyRecord(ctx, event.AsRecord(evt), decodedPayload)
//
//	// Query the DLQ:
//	result, _ := metaengine.ExecuteTyped[projections.DeadLetterQuery, projections.DeadLetterEntry](
//	    ctx, store, projections.DeadLetterQuery{CommandID: "01J..."},
//	)
package projections

import (
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// --- Query input types ---

// DeadLetterQuery queries the dead-letter queue. If CommandID is empty, the
// query returns all dead-lettered commands.
type DeadLetterQuery struct {
	CommandID string `json:"commandId,omitempty"`
}

// RetryCountQuery queries retry counts for commands.
type RetryCountQuery struct{}

// FailureLogQuery queries recent command failures.
type FailureLogQuery struct {
	Limit int `json:"limit,omitempty"`
}

// --- Result types ---

// DeadLetterEntry is the materialized view of a dead-lettered command. It is
// the value stored in the DLQ projection, keyed by command ID.
type DeadLetterEntry = commandlifecycle.DeadLetteredPayload

// --- Projection declarations ---

// DeadLetterQueue returns a metaengine projection declaration that folds
// command.dead-lettered events into a Map keyed by command ID. Query it to
// check whether a specific command is dead-lettered or to enumerate all
// dead-lettered commands.
//
// ADT: Map (key = command ID from CausationID, value = DeadLetteredPayload).
func DeadLetterQueue() metaengine.QueryDecl[DeadLetterQuery, DeadLetterEntry] {
	return metaengine.Query[DeadLetterQuery, DeadLetterEntry](
		"command_dlq",
		metaengine.OnRecordTyped(
			string(commandlifecycle.TypeDeadLettered),
			commandlifecycle.DeadLetteredPayload{}, //nolint:exhaustruct // type inference hint for OnRecordTyped
			func(rec record.Record, payload commandlifecycle.DeadLetteredPayload) (string, DeadLetterEntry) {
				return rec.MetaData.CausationID.String(), payload
			},
		),
	)
}

// RetryCount returns a metaengine projection declaration that folds
// command.retried events into a Counter. Each retried event increments the
// counter for its command by 1.
//
// ADT: Counter (key = command ID from CausationID, delta = +1 per retried event).
func RetryCount() metaengine.QueryDecl[RetryCountQuery, map[string]int64] {
	return metaengine.Query[RetryCountQuery, map[string]int64](
		"command_retry_count",
		metaengine.OnRecordTyped(
			string(commandlifecycle.TypeRetried),
			commandlifecycle.RetriedPayload{}, //nolint:exhaustruct // type inference hint for OnRecordTyped
			func(rec record.Record, _ commandlifecycle.RetriedPayload) metaengine.Delta {
				return metaengine.Delta{rec.MetaData.CausationID.String(): 1}
			},
		),
	)
}

// FailureLog returns a metaengine projection declaration that folds
// command.failed events into a Log. Query it to see recent failures in
// chronological order.
//
// ADT: Log (append FailedPayload per failed event).
func FailureLog() metaengine.QueryDecl[FailureLogQuery, []commandlifecycle.FailedPayload] {
	return metaengine.Query[FailureLogQuery, []commandlifecycle.FailedPayload](
		"command_failure_log",
		metaengine.OnRecordTyped(
			string(commandlifecycle.TypeFailed),
			commandlifecycle.FailedPayload{}, //nolint:exhaustruct // type inference hint for OnRecordTyped
			func(_ record.Record, payload commandlifecycle.FailedPayload) metaengine.Append {
				return metaengine.Append{Value: payload}
			},
		),
	)
}

// All returns all pre-built lifecycle projection declarations. Pass them to
// metaengine.Plan or system.DomainConfig.Projections.
func All() []any {
	return []any{
		DeadLetterQueue(),
		RetryCount(),
		FailureLog(),
	}
}
