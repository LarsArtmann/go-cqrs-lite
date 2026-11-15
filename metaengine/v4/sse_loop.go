package metaengine

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	sse "github.com/larsartmann/go-sse"
)

// sseDroppedEvents counts events evicted by the SSE ring's drop-old
// backpressure across the process lifetime. Eviction is BY DESIGN (a slow
// consumer must never stall the watcher), but it was invisible: the CI
// recurrence class "stream opened, later event never arrived" (2026-10-27,
// 2026-11-05, 2026-11-06 — :328) could not be separated from other loss
// mechanisms without this signal.
//
//nolint:gochecknoglobals // process-lifetime diagnostic counter, atomic
var sseDroppedEvents atomic.Int64

// SSEDroppedEvents reports how many events SSE ring buffers have evicted
// (drop-old backpressure for slow consumers) since process start. Tests
// assert a zero delta over fast-consumer windows: a drop there means an
// event was skipped that the consumer never had a chance to read, which
// reclassifies an "event never streamed" flake as backpressure in one
// query.
func SSEDroppedEvents() int64 { return sseDroppedEvents.Load() }

// forwardWithDropOld reads from src and forwards to dst with drop-old
// semantics: when dst is full, the oldest item is evicted to make room.
// An optional filter can drop items before forwarding (returns true = forward).
func forwardWithDropOld[T any](
	ctx context.Context,
	src <-chan T,
	dst chan T,
	filter func(T) bool,
) {
	defer close(dst)

	for {
		select {
		case <-ctx.Done():
			return
		case val, ok := <-src:
			if !ok {
				return
			}

			if filter != nil && !filter(val) {
				continue
			}

			select {
			case dst <- val:
			default:
				// Evict oldest to make room.
				select {
				case <-dst:
					sseDroppedEvents.Add(1)
				default:
				}

				select {
				case dst <- val:
				default:
				}
			}
		}
	}
}

// sseMainLoop runs the common SSE event loop: reads from buf, writes events
// via writeEvent, handles timeout and heartbeat. Returns nil on normal close
// (context cancel, channel close, timeout) or the write error.
func sseMainLoop[T any](
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	buf <-chan T,
	cfg SSEConfig,
	writeEvent func(http.ResponseWriter, T) error,
) error {
	var timer *time.Timer

	if cfg.Timeout > 0 {
		timer = time.NewTimer(cfg.Timeout)
		defer timer.Stop()
	}

	var heartbeat *time.Ticker

	if cfg.HeartbeatInterval > 0 {
		heartbeat = time.NewTicker(cfg.HeartbeatInterval)
		defer heartbeat.Stop()
	}

	var timerCh, heartbeatCh <-chan time.Time
	if timer != nil {
		timerCh = timer.C
	}

	if heartbeat != nil {
		heartbeatCh = heartbeat.C
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case val, ok := <-buf:
			if !ok {
				return nil
			}

			if err := writeEvent(w, val); err != nil {
				return err
			}

			flusher.Flush()

		case <-timerCh:
			return nil

		case <-heartbeatCh:
			if err := sse.WriteHeartbeat(w); err != nil {
				return err //nolint:wrapcheck
			}

			flusher.Flush()
		}
	}
}
