package event

import (
	"context"
	"time"
)

// deadlineCtx is a lightweight context.Context that carries a deadline
// without allocating a timer or goroutine (unlike context.WithDeadline).
type deadlineCtx struct{ deadline time.Time }

func (d *deadlineCtx) Deadline() (time.Time, bool) { return d.deadline, true }

func (d *deadlineCtx) Done() <-chan struct{} {
	if time.Now().After(d.deadline) {
		ch := make(chan struct{})
		close(ch)

		return ch
	}

	return nil
}

func (d *deadlineCtx) Err() error {
	if time.Now().After(d.deadline) {
		return context.DeadlineExceeded
	}

	return nil
}

func (*deadlineCtx) Value(any) any { return nil }
