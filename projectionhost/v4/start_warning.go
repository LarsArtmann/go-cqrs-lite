package projectionhost

import (
	"context"
	"errors"
	"sync"
	"time"
)

// errHostStopped is the cancellation cause Stop and ForceStop install on the
// worker context. Any other cause means the context passed to Host.Start was
// cancelled by its caller while the host is still open.
var errHostStopped = errors.New("projectionhost: host stopped")

// hostRun is one Start generation: the worker context, its cancel func and
// the latch that keeps the stale-projection warning to a single line per
// Start. A Stop→Start restart builds a fresh one.
type hostRun struct {
	host   *Host
	cancel context.CancelCauseFunc
	warned sync.Once
}

func newHostRun(parent context.Context, h *Host) (context.Context, *hostRun) {
	ctx, cancel := context.WithCancelCause(parent)

	//nolint:exhaustruct_v5 // warned starts unset
	return ctx, &hostRun{host: h, cancel: cancel}
}

// stop cancels the worker context with the host's own stop cause, so the
// workers exit without tripping the stale-projection warning.
func (r *hostRun) stop() {
	r.cancel(errHostStopped)
}

// runWorker staggers its start (see workerStartStaggerMs), runs one worker
// and reports a Start context that was cancelled underneath it. Every worker
// exit path funnels through here, so the check lives in one place instead of
// in each ctx.Done branch.
func (r *hostRun) runWorker(ctx context.Context, w *worker, delay int) {
	if delay > 0 {
		select {
		case <-time.After(time.Duration(delay) * time.Millisecond):
		case <-ctx.Done():
			r.warnIfStartContextCancelled(ctx)

			return
		}
	}

	w.run(ctx)
	r.warnIfStartContextCancelled(ctx)
}

// warnIfStartContextCancelled logs once per Start when the context passed to
// Host.Start was cancelled while the host was not stopped: workers then stay
// down and projections go stale with nothing else to show for it.
func (r *hostRun) warnIfStartContextCancelled(ctx context.Context) { // art-dupl:accept timers twin
	cause := context.Cause(ctx)
	if cause == nil || errors.Is(cause, errHostStopped) {
		return
	}

	r.warned.Do(func() {
		const hint = "Pass a context that lives as long as the host (context.WithoutCancel) and stop with Stop"
		r.host.opts.logger.Warn("projectionhost: projection workers stopped: the Start context was cancelled "+
			"while the host is not stopped; workers run only as long as that context, so projections stay "+
			"stale until Start is called again. "+hint, "cause", cause)
	})
}
