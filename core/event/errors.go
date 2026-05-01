package event

import "github.com/cockroachdb/errors"

// ErrVersionConflict is returned when there is a version conflict.
var ErrVersionConflict = errors.New("version conflict")

// ErrAggregateNotFound is returned when an aggregate is not found.
var ErrAggregateNotFound = errors.New("aggregate not found")

// ErrStoreClosed is returned when the event store is closed.
var ErrStoreClosed = errors.New("event store is closed")

// ErrBusClosed is returned when the event bus is closed.
var ErrBusClosed = errors.New("event bus is closed")

// ErrSnapshotNotFound is returned when a snapshot is not found.
var ErrSnapshotNotFound = errors.New("snapshot not found")

// ErrSnapshotStoreClosed is returned when the snapshot store is closed.
var ErrSnapshotStoreClosed = errors.New("snapshot store is closed")

// ErrNilProjection is returned when a nil projection is registered.
var ErrNilProjection = errors.New("event: nil projection")

// ErrNilCheckpointStore is returned when a nil checkpoint store is passed to NewInMemoryRunner.
var ErrNilCheckpointStore = errors.New("event: nil checkpoint store")

// ErrDuplicateProjection is returned when a projection with the same name is already registered.
var ErrDuplicateProjection = errors.New("event: duplicate projection")

// ErrNilOutbox is returned when a nil outbox is passed to NewOutboxPublisher.
var ErrNilOutbox = errors.New("event: nil outbox")

// ErrNilBus is returned when a nil bus is passed to NewOutboxPublisher.
var ErrNilBus = errors.New("event: nil bus")

// ErrAlreadyStarted is returned when OutboxPublisher.Start is called more than once.
var ErrAlreadyStarted = errors.New("event: outbox publisher already started")

// Family classifies an error's behavioral profile for automated handling.
type Family int

const (
	// Rejection indicates bad input, unauthorized access, or resource not found.
	// No state changed. Not retryable. Audience: requester only.
	Rejection Family = iota

	// Conflict indicates a version mismatch, duplicate creation, or state machine violation.
	// No state changed for requester. Not retryable. Audience: requester + subscribers.
	Conflict

	// Transient indicates a temporary infrastructure failure.
	// State unknown or no change. Retryable with backoff. Audience: requester + all clients.
	Transient

	// Corruption indicates the source of truth is damaged (unparseable payload, schema break).
	// Not self-healable. Not retryable. Audience: ops only.
	Corruption

	// Infrastructure indicates the system cannot serve (closed, nil deps, startup failure).
	// Not retryable. Audience: all clients + ops.
	Infrastructure
)

func (f Family) String() string {
	switch f {
	case Rejection:
		return "rejection"
	case Conflict:
		return "conflict"
	case Transient:
		return "transient"
	case Corruption:
		return "corruption"
	case Infrastructure:
		return "infrastructure"
	default:
		return "unknown"
	}
}

// Error is a classified error with a machine-readable code and family.
// Use errors.As to extract from wrapped chains.
type Error struct {
	Code    string
	Message string
	Family  Family
	cause   error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.cause }

// WithCause sets the underlying cause and returns the error for chaining.
func (e *Error) WithCause(cause error) *Error { e.cause = cause; return e }

// NewRejection creates a Rejection-classified error.
func NewRejection(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Rejection}
}

// NewConflict creates a Conflict-classified error.
func NewConflict(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Conflict}
}

// NewTransient creates a Transient-classified error (retryable).
func NewTransient(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Transient}
}

// NewCorruption creates a Corruption-classified error (source of truth damaged).
func NewCorruption(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Corruption}
}

// NewInfrastructure creates an Infrastructure-classified error (system cannot serve).
func NewInfrastructure(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Infrastructure}
}

// Classify returns the Family of an error by checking for *Error in the chain,
// then mapping known sentinel errors. Defaults to Transient for unknowns.
func Classify(err error) Family {
	var e *Error
	if errors.As(err, &e) {
		return e.Family
	}
	switch {
	case errors.Is(err, ErrVersionConflict):
		return Conflict
	case errors.Is(err, ErrAggregateNotFound),
		errors.Is(err, ErrSnapshotNotFound):
		return Rejection
	case errors.Is(err, ErrStoreClosed),
		errors.Is(err, ErrBusClosed),
		errors.Is(err, ErrSnapshotStoreClosed),
		errors.Is(err, ErrNilProjection),
		errors.Is(err, ErrNilCheckpointStore),
		errors.Is(err, ErrNilOutbox),
		errors.Is(err, ErrNilBus),
		errors.Is(err, ErrAlreadyStarted):
		return Infrastructure
	default:
		return Transient
	}
}

// IsRetryable returns true if the error is classified as Transient.
func IsRetryable(err error) bool { return Classify(err) == Transient }
