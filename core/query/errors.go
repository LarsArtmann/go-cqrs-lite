package query

import "github.com/cockroachdb/errors"

// ErrQueryNotSupported is returned when a query type is not supported.
var ErrQueryNotSupported = errors.New("query not supported")

// ErrQueryValidation is returned when query validation fails.
var ErrQueryValidation = errors.New("query validation failed")

// ErrDispatcherClosed is returned when the dispatcher is closed.
var ErrDispatcherClosed = errors.New("query dispatcher is closed")
