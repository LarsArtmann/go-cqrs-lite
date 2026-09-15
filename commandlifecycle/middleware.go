package commandlifecycle

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// attemptTracker counts processing attempts per command to detect retries
// and report accurate attempt counts in dead-lettered events. Bounded: under
// extreme concurrency an evicted entry only makes the next attempt report 1.
type attemptTracker struct {
	mu       sync.Mutex
	attempts *boundedMap[int]
	// rejected marks commands whose current attempt already emitted
	// command.rejected, so the outer middleware suppresses the redundant
	// terminal event (rejected commands are never dead-lettered).
	rejected *boundedMap[bool]
}

func newAttemptTracker() *attemptTracker {
	return &attemptTracker{
		mu:       sync.Mutex{},
		attempts: newBoundedMap[int](defaultCacheCapacity),
		rejected: newBoundedMap[bool](defaultCacheCapacity),
	}
}

func (t *attemptTracker) next(cmdID string) int {
	t.mu.Lock() //art-dupl:accept tiny accessor guards; extracting a generic helper hides intent
	defer t.mu.Unlock()

	v, _ := t.attempts.get(cmdID)
	t.attempts.put(cmdID, v+1)

	return v + 1
}

func (t *attemptTracker) get(cmdID string) int {
	t.mu.Lock() //art-dupl:accept tiny accessor guards; extracting a generic helper hides intent
	defer t.mu.Unlock()

	v, _ := t.attempts.get(cmdID)

	return v
}

func (t *attemptTracker) clear(cmdID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.attempts.delete(cmdID)
	t.rejected.delete(cmdID)
}

// markRejected records that the current attempt emitted command.rejected.
func (t *attemptTracker) markRejected(cmdID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.rejected.put(cmdID, true)
}

// take returns the attempt count and rejection marker for cmdID, clearing
// both entries. The terminal middleware consumes the attempt exactly once.
func (t *attemptTracker) take(cmdID string) (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	attempts, _ := t.attempts.get(cmdID)
	rejected, _ := t.rejected.get(cmdID)
	t.attempts.delete(cmdID)
	t.rejected.delete(cmdID)

	return attempts, rejected
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
func New(recorder *Recorder) (command.Middleware, command.Middleware) {
	tracker := newAttemptTracker()

	return outerMiddleware(recorder, tracker), attemptMiddleware(recorder, tracker)
}

// Middleware returns the outer lifecycle middleware for standalone use.
// Emits command.received, command.completed, and command.rejected or
// command.dead-lettered on terminal failure (classified via the recorder's
// rejection families).
//
// For full lifecycle tracking with retries, prefer [New] which returns both
// outer and attempt middleware with a shared tracker.
func Middleware(recorder *Recorder) command.Middleware {
	return outerMiddleware(recorder, nil)
}

// AttemptMiddleware returns the inner lifecycle middleware for standalone use.
// Emits command.failed or command.rejected and command.retried per processing
// attempt.
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
				alreadyRejected := false
				if tracker != nil {
					attempts, alreadyRejected = tracker.take(cmd.ID().String())
				}

				switch {
				case alreadyRejected:
					// The attempt middleware recorded command.rejected;
					// a rejection never retries and never dead-letters.
				case recorder.IsRejection(err):
					// Standalone outer middleware: classify here.
					_ = recorder.RecordRejected(ctx, cmd, err, attempts)
				default:
					_ = recorder.RecordDeadLettered(ctx, cmd, err, attempts)
				}
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
				if recorder.IsRejection(err) {
					tracker.markRejected(cmd.ID().String())
					_ = recorder.RecordRejected(ctx, cmd, err, attemptNum)
				} else {
					_ = recorder.RecordFailed(ctx, cmd, err, attemptNum)
				}

				return err
			}

			// Success is terminal for standalone use: without this, every
			// processed command leaves an entry here forever (the paired
			// outer middleware also clears, so this is idempotent there).
			tracker.clear(cmd.ID().String())

			return nil
		}
	}
}
