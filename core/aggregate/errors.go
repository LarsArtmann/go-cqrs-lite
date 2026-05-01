package aggregate

import "github.com/cockroachdb/errors"

var (
	ErrNilAggregateID     = errors.New("aggregate ID is required")
	ErrEmptyAggregateType = errors.New("aggregate type is required")
)
