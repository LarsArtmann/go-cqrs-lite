package event

import (
	"errors"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Re-export errorfamily types and functions so consumers use event.Family,
// event.Error, event.Classify, etc. without changing import paths.
type (
	// Family classifies an error's behavioral profile for automated handling.
	Family = errorfamily.Family

	// Error is a classified error with machine-readable code and family.
	Error = errorfamily.Error
)

// Family constants.
const (
	// Rejection indicates bad input, unauthorized access, or resource not found.
	Rejection = errorfamily.Rejection

	// Conflict indicates a version mismatch, duplicate creation, or state machine violation.
	Conflict = errorfamily.Conflict

	// Transient indicates a temporary infrastructure failure.
	Transient = errorfamily.Transient

	// Corruption indicates the source of truth is damaged (unparseable payload, schema break).
	Corruption = errorfamily.Corruption

	// Infrastructure indicates the system cannot serve (closed, nil deps, startup failure).
	Infrastructure = errorfamily.Infrastructure
)

// Classify forwards to errorfamily.Classify.
func Classify(err error) Family { return errorfamily.Classify(err) }

// IsRetryable reports whether the error is classified as Transient.
func IsRetryable(err error) bool { return errorfamily.IsRetryable(err) }

// RegisterClassification maps a sentinel error to a Family. Thread-safe.
func RegisterClassification(sentinel error, family Family) {
	errorfamily.RegisterClassification(sentinel, family)
}

// Constructors. Kept as aliases for backward compatibility.

// NewRejection creates a Rejection-classified error.
func NewRejection(code, msg string) *Error { return errorfamily.NewRejection(code, msg) }

// NewConflict creates a Conflict-classified error.
func NewConflict(code, msg string) *Error { return errorfamily.NewConflict(code, msg) }

// NewTransient creates a Transient-classified error.
func NewTransient(code, msg string) *Error { return errorfamily.NewTransient(code, msg) }

// NewCorruption creates a Corruption-classified error.
func NewCorruption(code, msg string) *Error { return errorfamily.NewCorruption(code, msg) }

// NewInfrastructure creates an Infrastructure-classified error.
func NewInfrastructure(code, msg string) *Error { return errorfamily.NewInfrastructure(code, msg) }

// Event domain sentinel errors.
var (
	ErrInvalidSnapshotInterval = errors.New("snapshot interval must be positive")
	ErrEmptyEventType          = errors.New("event type is required")
	ErrNilAggregateID          = errors.New("aggregate ID is required")
	ErrEmptyAggregateType      = errors.New("aggregate type is required")
	ErrVersionConflict         = errors.New("version conflict")
	ErrAggregateNotFound       = errors.New("aggregate not found")
	ErrStoreClosed             = errors.New("event store is closed")
	ErrBusClosed               = errors.New("event bus is closed")
	ErrSnapshotNotFound        = errors.New("snapshot not found")
	ErrSnapshotStoreClosed     = errors.New("snapshot store is closed")
	ErrNilProjection           = errors.New("event: nil projection")
	ErrNilCheckpointStore      = errors.New("event: nil checkpoint store")
	ErrDuplicateProjection     = errors.New("event: duplicate projection")
	ErrNilOutbox               = errors.New("event: nil outbox")
	ErrNilBus                  = errors.New("event: nil bus")
	ErrAlreadyStarted          = errors.New("event: outbox publisher already started")
	ErrProjectionPanicked      = errors.New("event: projection handler panicked")
)

//nolint:gochecknoinits
func init() {
	classifications := map[error]Family{
		ErrInvalidSnapshotInterval: Rejection,
		ErrEmptyEventType:          Rejection,
		ErrNilAggregateID:          Rejection,
		ErrEmptyAggregateType:      Rejection,
		ErrAggregateNotFound:       Rejection,
		ErrSnapshotNotFound:        Rejection,
		ErrVersionConflict:         Conflict,
		ErrDuplicateProjection:     Conflict,
		ErrStoreClosed:             Infrastructure,
		ErrBusClosed:               Infrastructure,
		ErrSnapshotStoreClosed:     Infrastructure,
		ErrNilProjection:           Infrastructure,
		ErrNilCheckpointStore:      Infrastructure,
		ErrNilOutbox:               Infrastructure,
		ErrNilBus:                  Infrastructure,
		ErrAlreadyStarted:          Infrastructure,
		ErrProjectionPanicked:      Corruption,
	}
	for sentinel, family := range classifications {
		RegisterClassification(sentinel, family)
	}
}
