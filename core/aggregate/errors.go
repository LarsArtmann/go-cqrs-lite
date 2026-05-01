package aggregate

import "github.com/cockroachdb/errors"

var (
	ErrNilAggregateID     = errors.New("aggregate ID is required")
	ErrEmptyAggregateType = errors.New("aggregate type is required")
	ErrNilStore           = errors.New("event store is required")
	ErrNilBus             = errors.New("event bus is required")
)
