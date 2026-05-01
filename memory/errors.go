package memory

import "github.com/cockroachdb/errors"

// ErrHandlerNil is returned when a nil handler is passed to Subscribe or SubscribeAll.
var ErrHandlerNil = errors.New("handler must not be nil")
