package sync

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrNilTimestampFunc is returned when NewLWWResolver is called with a nil timestamp function.
var ErrNilTimestampFunc = errorfamily.NewRejection(
	"sync.resolver.nil_timestamp_func",
	"NewLWWResolver requires a non-nil TimestampFunc",
)

// ErrInvalidOperationType is returned when an OperationType is not a valid known value.
var ErrInvalidOperationType = errorfamily.NewRejection(
	"sync.operation.invalid_type",
	"operation type is not valid",
)

// Clock order string constants for ClockOrder.String().
const (
	clockOrderBefore     = "before"
	clockOrderAfter      = "after"
	clockOrderConcurrent = "concurrent"
	clockOrderUnknown    = "unknown"
)

// NegativeCounterError is returned when a vector clock is created with a negative counter.
type NegativeCounterError struct {
	Node    NodeID
	Counter int64
}

func (e NegativeCounterError) Error() string {
	return fmt.Sprintf("negative counter %d for node %s", e.Counter, e.Node)
}
