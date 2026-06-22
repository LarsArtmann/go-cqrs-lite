package query

import (
	"errors"

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

func WrapCorruption(err error, code, msg string) *Error {
	return errorfamily.WrapCorruption(err, code, msg)
}

func WrapInfrastructure(wrappedErr error, code, msg string) *Error {
	return errorfamily.WrapInfrastructure(wrappedErr, code, msg)
}

func Wrapf(wrappedErr error, family Family, code, format string, args ...any) *Error {
	return errorfamily.Wrapf(wrappedErr, family, code, format, args...)
}

func Newf(family Family, code, format string, args ...any) *Error {
	return errorfamily.Newf(family, code, format, args...)
}

// Compose joins multiple errors into one, preserving all in the Unwrap chain.
//
//nolint:wrapcheck // re-export wrapper
func Compose(errs ...error) error {
	return errors.Join(errs...)
}

func ExitCode(err error) int { return errorfamily.ExitCode(err) }

// ErrHandlerNotFound is returned when no handler is registered for a query type.
var ErrHandlerNotFound = errorfamily.NewRejection(
	"query.handler_not_found",
	"no handler registered for query",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errorfamily.NewInfrastructure(
	"query.dispatcher_closed",
	"query dispatcher is closed",
)

// ErrEmptyQueryType is returned when a query is created with an empty type.
var ErrEmptyQueryType = errorfamily.NewRejection(
	"query.empty_query_type",
	"query type is required (got empty)",
)

// ErrTypeAssertion is returned when a query cannot be type-asserted to the expected type.
var ErrTypeAssertion = errorfamily.NewRejection(
	"query.type_assertion",
	"query type assertion failed",
)

// ErrStoreClosed is returned when the query store is closed.
var ErrStoreClosed = errorfamily.NewInfrastructure(
	"query.store_closed",
	"query store is closed",
)

// ErrQueryNotFound is returned when a query is not found in the store.
var ErrQueryNotFound = errorfamily.NewRejection(
	"query.not_found",
	"query not found",
)

// ErrDuplicateQuery is returned when a query with the same ID already exists.
var ErrDuplicateQuery = errorfamily.NewConflict(
	"query.duplicate",
	"query with this ID already exists",
)
