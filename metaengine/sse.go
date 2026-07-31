package metaengine

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SSEConfig configures Server-Sent Events streaming behavior.
type SSEConfig struct {
	// Timeout is the maximum duration the SSE stream stays open.
	// Zero means no timeout (stream until client disconnects).
	Timeout time.Duration

	// MaxBuffer is the maximum number of events buffered in the SSE
	// pipeline before applying drop-old semantics. When the buffer is
	// full, the oldest unread event is discarded to make room for the
	// newest. Zero defaults to 64.
	MaxBuffer int

	// HeartbeatInterval is how often to send a heartbeat comment
	// (":keepalive\n\n") to keep the connection alive through proxies.
	// Zero disables heartbeats.
	HeartbeatInterval time.Duration
}

// SSEOption configures an SSEConfig.
type SSEOption func(*SSEConfig)

// WithSSETimeout sets the maximum stream duration.
func WithSSETimeout(d time.Duration) SSEOption {
	return func(c *SSEConfig) { c.Timeout = d }
}

// WithSSEMaxBuffer sets the maximum number of buffered events.
// When the buffer fills, the oldest event is dropped (drop-old semantics).
func WithSSEMaxBuffer(n int) SSEOption {
	return func(c *SSEConfig) { c.MaxBuffer = n }
}

// WithSSEHeartbeat sets the interval for sending SSE heartbeat comments.
func WithSSEHeartbeat(d time.Duration) SSEOption {
	return func(c *SSEConfig) { c.HeartbeatInterval = d }
}

// ServeSSE streams Watcher notifications as Server-Sent Events over HTTP.
// Each value change is JSON-encoded and sent as an SSE "data" event.
//
// Backpressure: the SSE pipeline uses a ring buffer (default 64). When the
// client is slow and the buffer fills, the oldest unread event is dropped.
// A timeout option closes the stream after a maximum duration.
//
// Usage:
//
//	watcher := NewWatcher[UserView](store, "users")
//	defer watcher.Close()
//	http.HandleFunc("/events/users", func(w http.ResponseWriter, r *http.Request) {
//	    _ = metaengine.ServeSSE(w, r, watcher,
//	        metaengine.WithSSETimeout(30*time.Minute),
//	        metaengine.WithSSEHeartbeat(30*time.Second),
//	    )
//	})
//
// Clients connect with EventSource in the browser:
//
//	const es = new EventSource("/events/users");
//	es.onmessage = (e) => console.log(JSON.parse(e.data));
func ServeSSE[V any](
	w http.ResponseWriter,
	r *http.Request,
	watcher *Watcher[V],
	opts ...SSEOption,
) error {
	cfg := SSEConfig{MaxBuffer: 64}
	for _, opt := range opts {
		opt(&cfg)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("metaengine.ServeSSE: response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()

	var timer *time.Timer
	if cfg.Timeout > 0 {
		timer = time.NewTimer(cfg.Timeout)
		defer timer.Stop()
	}

	// Buffered pipeline: watcher values → SSE writer.
	// Drop-old semantics: if the buffer is full, the oldest value is discarded.
	buf := make(chan V, cfg.MaxBuffer)
	watchCh := watcher.Watch(ctx, nil)

	// Pump goroutine: read from watcher, push to buffer with drop-old.
	go func() {
		defer close(buf)

		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-watchCh:
				if !ok {
					return
				}

				select {
				case buf <- val:
				default:
					// Buffer full: drop oldest, push newest (drop-old semantics).
					select {
					case <-buf: // discard oldest
					default:
					}

					select {
					case buf <- val:
					default:
					}
				}
			}
		}
	}()

	var heartbeat *time.Ticker
	if cfg.HeartbeatInterval > 0 {
		heartbeat = time.NewTicker(cfg.HeartbeatInterval)
		defer heartbeat.Stop()
	}

	// Extract timer/heartbeat channels (nil channel = disabled in select).
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

			data, err := json.Marshal(val)
			if err != nil {
				continue
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return err
			}

			flusher.Flush()

		case <-timerCh:
			return nil // timeout reached

		case <-heartbeatCh:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

// Inspect returns a human-readable summary of all collections, their ADTs,
// read patterns, assigned engines, and complexity scores. Useful for CLI
// tools, debug endpoints, and startup diagnostics.
func (s *Store) Inspect() string {
	collections := s.Collections()

	if len(collections) == 0 {
		return "metaengine: no collections registered"
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "metaengine: %d collection(s)\n", len(collections))

	for _, c := range collections {
		fmt.Fprintf(
			&sb,
			"  %-20s  ADT=%-10s  pattern=%-20s  engine=%-15s  complexity=%s\n",
			c.Name, c.ADT, c.ReadPattern, c.EngineName, c.Complexity,
		)
	}

	return sb.String()
}

// InspectJSON returns a machine-readable JSON summary of all collections,
// suitable for API endpoints, monitoring tools, and structured logging.
func (s *Store) InspectJSON() ([]byte, error) {
	return json.Marshal(s.Collections())
}
