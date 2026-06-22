package command

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

func Compose(errs ...error) error {
	return errors.Join(errs...)
}

func ExitCode(err error) int { return errorfamily.ExitCode(err) }

// ErrHandlerNotFound is returned when no handler is registered for a command.
var ErrHandlerNotFound = errorfamily.NewRejection(
	"command.handler_not_found",
	"handler not found for command",
)

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errorfamily.NewInfrastructure(
	"command.dispatcher_closed",
	"command dispatcher is closed",
)

// ErrEmptyCommandType is returned by New when the command type is empty.
var ErrEmptyCommandType = errorfamily.NewRejection(
	"command.empty_command_type",
	"command type is required",
)

// ErrNilAggregateID is returned by New when the aggregate ID is zero.
var ErrNilAggregateID = errorfamily.NewRejection(
	"command.nil_aggregate_id",
	"aggregate ID is required",
)

// ErrTypeAssertion is returned when a command cannot be type-asserted to the expected type.
var ErrTypeAssertion = errorfamily.NewRejection(
	"command.type_assertion",
	"command type assertion failed",
)

// ErrEmptyAggregateType is returned when an aggregate type is empty.
var ErrEmptyAggregateType = errorfamily.NewRejection(
	"command.empty_aggregate_type",
	"aggregate type is required",
)

// ErrDuplicateCommand is returned when a command with the same ID already exists.
var ErrDuplicateCommand = errorfamily.NewConflict(
	"command.duplicate",
	"command with this ID already exists",
)

// ErrCommandNotFound is returned when a command is not found.
var ErrCommandNotFound = errorfamily.NewRejection(
	"command.not_found",
	"command not found",
)

// ErrStoreClosed is returned when the command store is closed.
var ErrStoreClosed = errorfamily.NewInfrastructure(
	"command.store_closed",
	"command store is closed",
)
