package commandlifecycle

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// attemptTracker counts processing attempts per command to detect retries
// and report accurate attempt counts in dead-lettered events.
type attemptTracker struct {
	mu       sync.Mutex
	attempts map[string]int
}

func newAttemptTracker() *attemptTracker {
	return &attemptTracker{attempts: make(map[string]int)}
}

func (t *attemptTracker) next(cmdID string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.attempts[cmdID]++

	return t.attempts[cmdID]
}

func (t *attemptTracker) get(cmdID string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.attempts[cmdID]
}

func (t *attemptTracker) clear(cmdID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.attempts, cmdID)
}

// New returns a pair of middleware — outer and attempt — that together produce
// the full command lifecycle event stream. They share an attempt tracker so
// dead-lettered events carry accurate attempt counts.
//
// Wire them around retry middleware:
//
//	outer, attempt := commandlifecycle.New(recorder)
//	dispatcher.Use(
//	    outer,                              // received, completed, dead-lettered
//	    middleware.CommandRetry(config),    // handles retries
//	    attempt,                            // failed, retried (per attempt)
//	)
//
// Without retry middleware, use only the outer:
//
//	outer, _ := commandlifecycle.New(recorder)
//	dispatcher.Use(outer)
func New(recorder *Recorder) (outer, attempt command.Middleware) {
	tracker := newAttemptTracker()

	return outerMiddleware(recorder, tracker), attemptMiddleware(recorder, tracker)
}

// Middleware returns the outer lifecycle middleware for standalone use.
// Emits command.received, command.completed, and command.dead-lettered.
//
// For full lifecycle tracking with retries, prefer [New] which returns both
// outer and attempt middleware with a shared tracker.
func Middleware(recorder *Recorder) command.Middleware {
	return outerMiddleware(recorder, nil)
}

// AttemptMiddleware returns the inner lifecycle middleware for standalone use.
// Emits command.failed and command.retried per processing attempt.
//
// For full lifecycle tracking, prefer [New] which returns both outer and
// attempt middleware with a shared tracker.
func AttemptMiddleware(recorder *Recorder) command.Middleware {
	return attemptMiddleware(recorder, newAttemptTracker())
}

func outerMiddleware(recorder *Recorder, tracker *attemptTracker) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			_ = recorder.RecordReceived(ctx, cmd)

			err := next(ctx, cmd)
			if err != nil {
				attempts := 1
				if tracker != nil {
					attempts = tracker.get(cmd.ID().String())
					tracker.clear(cmd.ID().String())
				}

				_ = recorder.RecordDeadLettered(ctx, cmd, err, attempts)
			} else {
				if tracker != nil {
					tracker.clear(cmd.ID().String())
				}

				_ = recorder.RecordCompleted(ctx, cmd)
			}

			return err
		}
	}
}

func attemptMiddleware(recorder *Recorder, tracker *attemptTracker) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			attemptNum := tracker.next(cmd.ID().String())

			if attemptNum > 1 {
				_ = recorder.RecordRetried(ctx, cmd, attemptNum-1)
			}

			err := next(ctx, cmd)
			if err != nil {
				_ = recorder.RecordFailed(ctx, cmd, err, attemptNum)

				return err
			}

			return nil
		}
	}
}
