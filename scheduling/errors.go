package scheduling

import "errors"

// ErrEmptyTimerID is returned by [ParseTimerID] for empty input. Timer IDs
// key idempotent scheduling — an empty key cannot address a timer.
var ErrEmptyTimerID = errors.New("scheduling: timer ID must be non-empty")
