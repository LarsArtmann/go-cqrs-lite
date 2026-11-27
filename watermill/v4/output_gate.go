package watermill

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// outputGate guards an adapter's shared output channel against its Close.
//
// The handlers an adapter registers on the cqrs bus are never unsubscribed
// (the bus has no unsubscribe), so they keep running for every publish after
// Close. A handler that then selects on a closed output channel and a closed
// close channel picks a ready case at random and panics the publisher with
// "send on closed channel". The gate refuses offers once Close started and
// lets Close wait for in-flight sends, so the channel is only closed when no
// sender can reach it.
type outputGate[M any] struct {
	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
	sends     sync.WaitGroup
}

// offer hands msg to output. It returns [event.ErrBusClosed] once Close
// started (or releases a sender parked on a full channel) and ctx.Err() when
// the publish context ends first; callers wrap the error with their own code.
func (g *outputGate[M]) offer(
	ctx context.Context,
	msg M,
	output chan<- M,
	closeCh <-chan struct{},
) error {
	g.mu.RLock()

	if g.closed {
		g.mu.RUnlock()

		return event.ErrBusClosed
	}

	// Registered under the same lock Close flips, so Close's Wait always
	// observes this sender.
	g.sends.Add(1)
	defer g.sends.Done()

	g.mu.RUnlock()

	select {
	case output <- msg:
		return nil
	case <-closeCh:
		return event.ErrBusClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

// close stops new offers, releases parked senders through closeCh and closes
// output only after every in-flight send returned. Messages already buffered
// stay readable. Safe to call more than once.
func (g *outputGate[M]) close(closeCh chan struct{}, output chan M) {
	g.closeOnce.Do(func() {
		g.mu.Lock()
		g.closed = true
		g.mu.Unlock()

		close(closeCh)
		g.sends.Wait()
		close(output)
	})
}
