package sync

import errorfamily "github.com/larsartmann/go-error-family"

// ErrNilTimestampFunc is returned when NewLWWResolver is called with a nil timestamp function.
var ErrNilTimestampFunc = errorfamily.NewRejection(
	"sync.resolver.nil_timestamp_func",
	"NewLWWResolver requires a non-nil TimestampFunc",
)
