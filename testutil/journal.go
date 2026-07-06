package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// DelayedJournal wraps an [event.SeekableJournal] and blocks ReadFrom for the
// configured delay, respecting context cancellation. Useful for testing
// timeout behaviour in replay/catch-up paths without slow CI flakiness.
//
// All other methods (ReadAll, Save, Load, etc.) are forwarded unchanged via
// the embedded [event.SeekableJournal].
type DelayedJournal struct {
	event.SeekableJournal

	// Delay is how long ReadFrom blocks before delegating to the wrapped journal.
	Delay time.Duration
}

// NewDelayedJournal wraps inner with a ReadFrom delay of d.
func NewDelayedJournal(inner event.SeekableJournal, d time.Duration) *DelayedJournal {
	return &DelayedJournal{SeekableJournal: inner, Delay: d}
}

// ReadFrom blocks for Delay (or until ctx is cancelled), then forwards to the
// wrapped journal. A cancelled context returns ctx.Err() immediately.
func (j *DelayedJournal) ReadFrom(
	ctx context.Context,
	after id.EventID,
	limit int,
) ([]event.Event, error) {
	select {
	case <-time.After(j.Delay):
	case <-ctx.Done():
		return nil, fmt.Errorf(
			"delayed journal cancelled: %w",
			ctx.Err(),
		)
	}

	return j.SeekableJournal.ReadFrom(ctx, after, limit) //nolint:wrapcheck // direct delegation
}
