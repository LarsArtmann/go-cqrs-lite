// Package commandlifecycle implements command lifecycle tracking as event
// streams (ADR-0117).
//
// Commands are immutable intents with no status field. Their lifecycle —
// received, rejected, failed, retried, dead-lettered, completed — is tracked
// via events appended to a per-command lifecycle stream. Dead-letter queues,
// retry counts, failure logs, and rejection logs emerge as projections over
// these event streams.
//
// # Stream Model
//
// Each command has two streams:
//
//	Stream: Command/<cmd-id>          — the immutable command intent
//	Stream: CommandLifecycle/<cmd-id> — lifecycle events for that command
//
// Lifecycle events carry CausationID = the command ID, linking them back to
// the command they describe.
//
// # Usage
//
//	recorder := commandlifecycle.NewRecorder(eventStore)
//
//	outer, attempt := commandlifecycle.New(recorder)
//	dispatcher.Use(
//	    outer,                              // emits received, completed, dead-lettered, rejected
//	    middleware.CommandRetry(config),    // handles retries
//	    attempt,                            // emits failed, rejected, retried (per attempt)
//	)
//
// For querying (DLQ, retry counts, failure logs), use the pre-built
// projection declarations from the commandlifecycle/projections sub-module.
package commandlifecycle

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// StreamTypeCommandLifecycle is the stream type for command lifecycle event
// streams. Each command gets its own lifecycle stream:
// CommandLifecycle/<cmd-id>.
const StreamTypeCommandLifecycle id.StreamType = id.StreamType("CommandLifecycle")

// Lifecycle event types. These are the event type strings appended to
// CommandLifecycle streams.
const (
	// TypeReceived is emitted when the server accepts a command for processing.
	TypeReceived event.Type = event.Type("command.received")

	// TypeFailed is emitted when a single processing attempt fails with a
	// retryable error.
	TypeFailed event.Type = event.Type("command.failed")

	// TypeRejected is emitted when a processing attempt fails with a business
	// rejection: an error whose errorfamily classification is in the
	// recorder's rejection families (default: Rejection and Conflict, see
	// WithRejectionFamilies). A rejected command changed no state, will not
	// be retried (errorfamily.IsRetryable is false for rejection families),
	// and is not dead-lettered — the audit trail can therefore distinguish
	// "rejected by business rule" (command.rejected) from "broke"
	// (command.failed) by event type alone.
	TypeRejected event.Type = event.Type("command.rejected")

	// TypeRetried is emitted before each retry attempt (after the first).
	TypeRetried event.Type = event.Type("command.retried")

	// TypeDeadLettered is emitted when all retry attempts are exhausted.
	TypeDeadLettered event.Type = event.Type("command.dead-lettered")

	// TypeCompleted is emitted when the command is processed successfully.
	TypeCompleted event.Type = event.Type("command.completed")
)

// CommandKey is a named string type used for command IDs in lifecycle event
// payloads. The distinct type enables unambiguous key extraction in
// metaengine projections that join events by command ID (e.g. ProcessingTime).
type CommandKey string

// ReceivedPayload is the payload for command.received events.
type ReceivedPayload struct {
	// CommandID is the unique identifier of the command.
	CommandID CommandKey `json:"commandId"`

	// CommandType is the type of the command that was received.
	CommandType string `json:"commandType"`

	// CommandStreamID is the target stream of the command.
	CommandStreamID string `json:"commandStreamId"`

	// ReceivedAt is when the server received the command.
	ReceivedAt time.Time `json:"receivedAt"`
}

// FailedPayload is the payload for command.failed events.
type FailedPayload struct {
	// CommandID is the unique identifier of the command. Empty on payloads
	// recorded before v4.4 (projections key off the stream ref instead).
	CommandID CommandKey `json:"commandId,omitempty"`

	// CommandType is the type of the command that failed.
	CommandType string `json:"commandType"`

	// Error is the error message from the failed attempt.
	Error string `json:"error"`

	// Attempt is the 1-indexed attempt number that failed.
	Attempt int `json:"attempt"`

	// FailedAt is when the attempt failed.
	FailedAt time.Time `json:"failedAt"`
}

// RejectedPayload is the payload for command.rejected events. It carries the
// errorfamily classification stamped at rejection time, so audit consumers
// can tell "rejected by business rule" (family rejection/conflict) from
// "broke" (command.failed) without re-classifying error text.
type RejectedPayload struct {
	// CommandID is the unique identifier of the command.
	CommandID CommandKey `json:"commandId"`

	// CommandType is the type of the command that was rejected.
	CommandType string `json:"commandType"`

	// Error is the rejection error message.
	Error string `json:"error"`

	// Family is the errorfamily classification ("rejection", "conflict", …).
	Family string `json:"family"`

	// ErrorCode is the machine-readable error code, if the error chain
	// declares one (errorfamily.Code). Empty when uncoded.
	ErrorCode string `json:"errorCode,omitempty"`

	// Attempt is the 1-indexed attempt number that was rejected. Always 1
	// for default wiring: rejections are non-retryable, so there is no
	// subsequent attempt.
	Attempt int `json:"attempt"`

	// RejectedAt is when the command was rejected.
	RejectedAt time.Time `json:"rejectedAt"`
}

// RetriedPayload is the payload for command.retried events.
type RetriedPayload struct {
	// CommandType is the type of the command being retried.
	CommandType string `json:"commandType"`

	// Attempt is the 1-indexed retry number (1 = first retry, i.e., 2nd attempt).
	Attempt int `json:"attempt"`

	// RetriedAt is when the retry was scheduled.
	RetriedAt time.Time `json:"retriedAt"`
}

// DeadLetteredPayload is the payload for command.dead-lettered events.
type DeadLetteredPayload struct {
	// CommandType is the type of the command that was dead-lettered.
	CommandType string `json:"commandType"`

	// Error is the final error message after all attempts.
	Error string `json:"error"`

	// Attempts is the total number of attempts made.
	Attempts int `json:"attempts"`

	// DeadLetteredAt is when the command was moved to the dead-letter queue.
	DeadLetteredAt time.Time `json:"deadLetteredAt"`
}

// CompletedPayload is the payload for command.completed events.
type CompletedPayload struct {
	// CommandID is the unique identifier of the command.
	CommandID CommandKey `json:"commandId"`

	// CommandType is the type of the command that completed.
	CommandType string `json:"commandType"`

	// CompletedAt is when the command finished processing.
	CompletedAt time.Time `json:"completedAt"`
}

// LifecycleStreamRef returns the event stream ref for a command's lifecycle
// events. The stream is keyed by the command ID:
// CommandLifecycle/<cmd-id>.
func LifecycleStreamRef(cmd command.Command) id.StreamRef {
	return id.NewStreamRef(
		StreamTypeCommandLifecycle,
		id.StreamIDFrom(cmd.ID()),
	)
}
