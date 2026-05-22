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

// Clock order string constants for ClockOrder.String().
const (
	clockOrderBefore     = "before"
	clockOrderAfter      = "after"
	clockOrderConcurrent = "concurrent"
)

// NegativeCounterError is returned when a vector clock is created with a negative counter.
type NegativeCounterError struct {
	Node    NodeID
	Counter int64
}

func (e NegativeCounterError) Error() string {
	return fmt.Sprintf("negative counter %d for node %s", e.Counter, e.Node)
}
