package metaengine

import (
	"context"
	"net/http"
	"time"

	sse "github.com/larsartmann/go-sse"
)

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
