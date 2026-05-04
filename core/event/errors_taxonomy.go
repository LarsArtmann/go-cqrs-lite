package event

import (
	"fmt"
	"sync"

	"github.com/cockroachdb/errors"
)

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

// Error returns the human-readable error message.
func (e *Error) Error() string { return e.Message }

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error { return e.cause }

// Is reports whether this error matches another *Error by Code and Family.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}

	return e.Code == t.Code && e.Family == t.Family
}

// Format implements fmt.Formatter for verbose error output.
// Use %+v for family:code: message with cause chain.
func (e *Error) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			_, _ = fmt.Fprintf(f, "%s:%s: %s", e.Family, e.Code, e.Message)
			if e.cause != nil {
				_, _ = fmt.Fprintf(f, "\ncaused by: %+v", e.cause)
			}

			return
		}

		fallthrough
	default:
		_, _ = fmt.Fprint(f, e.Message)
	}
}

// WithCause sets the underlying cause and returns the error for chaining.
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause

	return e
}

// NewRejection creates a Rejection-classified error.
func NewRejection(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Rejection, cause: nil}
}

// NewConflict creates a Conflict-classified error.
func NewConflict(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Conflict, cause: nil}
}

// NewTransient creates a Transient-classified error (retryable).
func NewTransient(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Transient, cause: nil}
}

// NewCorruption creates a Corruption-classified error (source of truth damaged).
func NewCorruption(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Corruption, cause: nil}
}

// NewInfrastructure creates an Infrastructure-classified error (system cannot serve).
func NewInfrastructure(code, msg string) *Error {
	return &Error{Code: code, Message: msg, Family: Infrastructure, cause: nil}
}

// Classify returns the Family of an error by checking for *Error in the chain,
// then mapping known sentinel errors from the event package and any registered
// external sentinels. Returns Rejection for nil errors. Defaults to Transient for unknowns.
//
// External packages (command, query, aggregate, projection, storage, decider) can register
// their own sentinel classifications via RegisterClassification.
func Classify(err error) Family {
	if err == nil {
		return Rejection
	}

	var e *Error
	if errors.As(err, &e) {
		return e.Family
	}

	if family, ok := lookupRegistered(err); ok {
		return family
	}

	switch {
	case errors.Is(err, ErrVersionConflict),
		errors.Is(err, ErrDuplicateProjection):
		return Conflict
	case errors.Is(err, ErrAggregateNotFound),
		errors.Is(err, ErrSnapshotNotFound),
		errors.Is(err, ErrInvalidSnapshotInterval),
		errors.Is(err, ErrEmptyEventType),
		errors.Is(err, ErrNilAggregateID),
		errors.Is(err, ErrEmptyAggregateType):
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
	case errors.Is(err, ErrProjectionPanicked):
		return Corruption
	default:
		return Transient
	}
}

var classifier = struct { //nolint:gochecknoglobals
	mu   sync.RWMutex
	maps map[error]Family
}{
	mu:   sync.RWMutex{},
	maps: make(map[error]Family),
}

// RegisterClassification maps a sentinel error to a Family.
// Thread-safe. Call from init() in external packages:
//
//	func init() {
//	    event.RegisterClassification(ErrHandlerNotFound, event.Rejection)
//	}
func RegisterClassification(sentinel error, family Family) {
	classifier.mu.Lock()
	defer classifier.mu.Unlock()

	classifier.maps[sentinel] = family
}

func lookupRegistered(err error) (Family, bool) {
	classifier.mu.RLock()
	defer classifier.mu.RUnlock()

	for sentinel, family := range classifier.maps {
		if errors.Is(err, sentinel) {
			return family, true
		}
	}

	return Rejection, false
}

// IsRetryable returns true if the error is classified as Transient.
func IsRetryable(err error) bool { return Classify(err) == Transient }
