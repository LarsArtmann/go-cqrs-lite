package query

import "github.com/cockroachdb/errors"

var (
	ErrQueryNotSupported = errors.New("query not supported")
	ErrQueryValidation   = errors.New("query validation failed")
	ErrDispatcherClosed  = errors.New("query dispatcher is closed")
)
