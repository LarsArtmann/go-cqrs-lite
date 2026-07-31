package metaengine

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
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

	// ReplayLimit caps the number of events replayed on reconnection.
	// Zero means replay all available events from the journal.
	// Only applies when the watcher has a replay journal (WithReplay).
	ReplayLimit int
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

// WithSSEReplayLimit caps the number of events replayed from the journal on
// reconnection. Zero (default) replays all available events. Use a bounded
// limit to prevent slow clients from receiving a huge backlog on reconnect.
// Only applies when the watcher has a replay journal (WithReplay).
func WithSSEReplayLimit(n int) SSEOption {
	return func(c *SSEConfig) { c.ReplayLimit = n }
}

// ServeSSE streams Watcher notifications as Server-Sent Events over HTTP.
// Each value change is JSON-encoded and sent as an SSE "data" event.
//
// Backpressure: the SSE pipeline uses a ring buffer (default 64). When the
// client is slow and the buffer fills, the oldest unread event is dropped.
// A timeout option closes the stream after a maximum duration.
//
// Reconnection: when the watcher has a replay journal (attached via
// [Watcher.WithReplay]), ServeSSE writes an "id: <seq>" field on every event.
// On reconnect, clients send the "Last-Event-ID" header with the last sequence
// number they received. ServeSSE replays missed events from the journal
// before switching to live streaming, with dedup to handle the replay→live
// overlap. Use [WithSSEReplayLimit] to cap the number of replayed events.
//
// Usage:
//
//	watcher := NewWatcher[UserView](store, "users")
//	replay := watcher.WithReplay(1000) // enable reconnection
//	defer watcher.Close()
//	http.HandleFunc("/events/users", func(w http.ResponseWriter, r *http.Request) {
//	    _ = metaengine.ServeSSE(w, r, watcher,
//	        metaengine.WithSSETimeout(30*time.Minute),
//	        metaengine.WithSSEHeartbeat(30*time.Second),
//	        metaengine.WithSSEReplayLimit(500),
//	    )
//	})
//
// Clients connect with EventSource in the browser — Last-Event-ID is handled
// automatically:
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
		return errSSENoFlusher
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	replay := watcher.Replay()
	if replay != nil {
		return serveSSEReplay(w, r, flusher, watcher, replay, cfg)
	}

	return serveSSEPlain(w, r, flusher, watcher, cfg)
}

// serveSSEPlain is the non-reconnection path: no id field, no dedup.
func serveSSEPlain[V any](
	w http.ResponseWriter,
	r *http.Request,
	flusher http.Flusher,
	watcher *Watcher[V],
	cfg SSEConfig,
) error {
	ctx := r.Context()

	var timer *time.Timer
	if cfg.Timeout > 0 {
		timer = time.NewTimer(cfg.Timeout)
		defer timer.Stop()
	}

	buf := make(chan V, cfg.MaxBuffer)
	watchCh := watcher.Watch(ctx, nil)

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
					select {
					case <-buf:
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
			return nil

		case <-heartbeatCh:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return err
			}

			flusher.Flush()
		}
	}
}

// replayMissedEvents writes events from the journal that the client missed
// (seq > afterSeq) and returns the set of sequence numbers that were sent,
// for deduplication during the live phase.
func replayMissedEvents[V any](
	w http.ResponseWriter,
	r *http.Request,
	replay *SSEReplay[V],
	cfg SSEConfig,
) (map[uint64]struct{}, error) {
	var afterSeq uint64

	if lastID := r.Header.Get("Last-Event-ID"); lastID != "" {
		if parsed, err := strconv.ParseUint(lastID, 10, 64); err == nil {
			afterSeq = parsed
		}
	}

	replayed := replay.Replay(afterSeq)
	if cfg.ReplayLimit > 0 && len(replayed) > cfg.ReplayLimit {
		replayed = replayed[:cfg.ReplayLimit]
	}

	replayedSeqs := make(map[uint64]struct{}, len(replayed))

	for _, sv := range replayed {
		data, err := json.Marshal(sv.Value)
		if err != nil {
			continue
		}

		if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", sv.Seq, data); err != nil { //nolint:wrapcheck // SSE write
			return nil, err
		}

		replayedSeqs[sv.Seq] = struct{}{}
	}

	return replayedSeqs, nil
}

// serveSSEReplay is the reconnection path: replays missed events from the
// journal on reconnect, then switches to live with dedup.
func serveSSEReplay[V any](
	w http.ResponseWriter,
	r *http.Request,
	flusher http.Flusher,
	watcher *Watcher[V],
	replay *SSEReplay[V],
	cfg SSEConfig,
) error {
	ctx := r.Context()

	// Phase 1: Replay missed events from the journal.
	replayedSeqs, err := replayMissedEvents(w, r, replay, cfg)
	if err != nil {
		return err
	}

	if len(replayedSeqs) > 0 {
		flusher.Flush()
	}

	// Phase 2: Live streaming with dedup (skip events already replayed).
	var timer *time.Timer
	if cfg.Timeout > 0 {
		timer = time.NewTimer(cfg.Timeout)
		defer timer.Stop()
	}

	buf := make(chan SeqValue[V], cfg.MaxBuffer)
	watchCh := watcher.WatchWithSeq(ctx, nil)

	go func() {
		defer close(buf)

		for {
			select {
			case <-ctx.Done():
				return
			case sv, ok := <-watchCh:
				if !ok {
					return
				}

				// Dedup: skip events that were already replayed.
				if sv.Seq > 0 {
					if _, dup := replayedSeqs[sv.Seq]; dup {
						continue
					}
				}

				select {
				case buf <- sv:
				default:
					select {
					case <-buf:
					default:
					}

					select {
					case buf <- sv:
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

		case sv, ok := <-buf:
			if !ok {
				return nil
			}

			data, err := json.Marshal(sv.Value)
			if err != nil {
				continue
			}

			if _, err := fmt.Fprintf(w, "id: %d\ndata: %s\n\n", sv.Seq, data); err != nil { //nolint:wrapcheck // SSE write
				return err
			}

			flusher.Flush()

		case <-timerCh:
			return nil

		case <-heartbeatCh:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil { //nolint:wrapcheck // SSE write
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
