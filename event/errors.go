package event

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

type (
	Family = errorfamily.Family
	Error  = errorfamily.Error
)

const (
	Rejection      = errorfamily.Rejection
	Conflict       = errorfamily.Conflict
	Transient      = errorfamily.Transient
	Corruption     = errorfamily.Corruption
	Infrastructure = errorfamily.Infrastructure
)

func Classify(err error) Family  { return errorfamily.Classify(err) }
func IsRetryable(err error) bool { return errorfamily.IsRetryable(err) }

func NewRejection(code, msg string) *Error {
	return errorfamily.NewRejection(code, msg)
}

func NewConflict(code, msg string) *Error { return errorfamily.NewConflict(code, msg) }

func NewTransient(code, msg string) *Error {
	return errorfamily.NewTransient(code, msg)
}

func NewCorruption(code, msg string) *Error {
	return errorfamily.NewCorruption(code, msg)
}

func NewInfrastructure(code, msg string) *Error {
	return errorfamily.NewInfrastructure(code, msg)
}

func Wrap(err error, family Family, code, msg string) *Error {
	return errorfamily.Wrap(err, family, code, msg)
}

func WrapRejection(err error, code, msg string) *Error {
	return errorfamily.WrapRejection(err, code, msg)
}

func WrapConflict(err error, code, msg string) *Error {
	return errorfamily.WrapConflict(err, code, msg)
}

func WrapTransient(err error, code, msg string) *Error {
	return errorfamily.WrapTransient(err, code, msg)
}

func WrapCorruption(err error, code, msg string) *Error {
	return errorfamily.WrapCorruption(err, code, msg)
}

func WrapInfrastructure(err error, code, msg string) *Error {
	return errorfamily.WrapInfrastructure(err, code, msg)
}

func Wrapf(err error, family Family, code, format string, args ...any) *Error {
	return errorfamily.Wrapf(err, family, code, format, args...)
}

func Newf(family Family, code, format string, args ...any) *Error {
	return errorfamily.Newf(family, code, format, args...)
}

func WithContext(err *Error, key, value string) *Error {
	return err.WithContext(key, value)
}

func ExitCode(err error) int    { return errorfamily.ExitCode(err) }
func HandleError(err error) int { return errorfamily.HandleError(err) }
func HandleErrorDetailed(err error) *HandleResult {
	return errorfamily.HandleErrorDetailed(err)
}

func RegisterTemplate(code string, tmpl MessageTemplate) {
	errorfamily.RegisterTemplate(code, tmpl)
}

func RegisterClassification(sentinel error, family Family) {
	errorfamily.RegisterClassification(sentinel, family)
}

func RegisterClassifications(classifications map[error]Family) {
	errorfamily.RegisterClassifications(classifications)
}

type (
	HandleResult      = errorfamily.HandleResult
	HandleConfig      = errorfamily.HandleConfig
	MessageTemplate   = errorfamily.MessageTemplate
	DiagnosticFinding = errorfamily.DiagnosticFinding
	DiagnosticFunc    = errorfamily.DiagnosticFunc
)

var (
	ErrEmptyEventType     = NewRejection("event.empty_event_type", "event type is required")
	ErrNilAggregateID     = NewRejection("event.nil_aggregate_id", "aggregate ID is required")
	ErrEmptyAggregateType = NewRejection(
		"event.empty_aggregate_type",
		"aggregate type is required",
	)
	ErrVersionNotPositive = NewRejection(
		"event.version_not_positive",
		"version must be positive",
	)
	ErrNilPayload           = NewRejection("event.nil_payload", "payload is required")
	ErrMismatchedEventCount = NewRejection(
		"event.mismatched_event_count",
		"event types and payloads count must match",
	)
	ErrVersionConflict   = NewConflict("event.version_conflict", "version conflict")
	ErrAggregateNotFound = NewRejection("event.aggregate_not_found", "aggregate not found")
	ErrStoreClosed       = NewInfrastructure("event.store_closed", "event store is closed")
	ErrBusClosed         = NewInfrastructure("event.bus_closed", "event bus is closed")
	ErrNilBus            = NewInfrastructure("event.nil_bus", "nil bus")
)
