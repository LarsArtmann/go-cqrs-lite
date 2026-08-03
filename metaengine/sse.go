package metaengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
	sse "github.com/larsartmann/go-sse"
)

// SSEConfig configures Server-Sent Events streaming behavior.
//
// This SSE implementation watches a metaengine Store collection for changes
// (collection-watch + replay from journal). For event-bus-to-client SSE
// (bridging an event.Bus to HTTP clients), see transport/http.SSEBroker.
// The two implementations serve different layers: metaengine SSE streams
// materialized query results (read-model push), while transport/http SSE
// streams raw domain events (event-bus push).
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

	sse.SetHeaders(w)
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

	buf := make(chan V, cfg.MaxBuffer)
	watchCh := watcher.Watch(ctx, nil)

	go forwardWithDropOld(ctx, watchCh, buf, nil)

	return sseMainLoop(ctx, w, flusher, buf, cfg, writePlainSSEEvent[V])
}

// writePlainSSEEvent marshals val as JSON and writes a plain SSE data event.
// Marshal failures are silently skipped to keep the stream alive.
// Wire-format serialization is delegated to [sse.WriteEvent] (ADR-0097).
func writePlainSSEEvent[V any](w http.ResponseWriter, val V) error {
	data, err := json.Marshal(val)
	if err != nil {
		return nil //nolint:nilerr // skip unmarshalable, keep stream alive
	}

	return sse.WriteEvent(w, sse.Event{Data: string(data)})
}

// serveSSEReplay is the reconnection path: subscribes first (to buffer live
// events during replay), then replays missed events from the journal, then
// switches to live streaming with dedup to skip events that overlap.
func serveSSEReplay[V any](
	w http.ResponseWriter,
	r *http.Request,
	flusher http.Flusher,
	watcher *Watcher[V],
	replay *SSEReplay[V],
	cfg SSEConfig,
) error {
	ctx := r.Context()

	// Phase 0: Subscribe FIRST to buffer live events arriving during replay.
	buf := make(chan SeqValue[V], cfg.MaxBuffer)
	watchCh := watcher.WatchWithSeq(ctx, nil)

	// Phase 1: Replay missed events from the journal.
	dedupRing, err := replayMissedEvents(w, r, replay, cfg)
	if err != nil {
		return err
	}

	if dedupRing.Len() > 0 {
		flusher.Flush()
	}

	// Phase 2: Live streaming with dedup (skip events already replayed).
	go forwardWithDropOld(ctx, watchCh, buf, func(sv SeqValue[V]) bool {
		return sv.Seq == 0 || !dedupRing.Has(strconv.FormatUint(sv.Seq, 10))
	})

	return sseMainLoop(ctx, w, flusher, buf, cfg, writeReplaySSEEvent[V])
}

// writeReplaySSEEvent marshals item.Value as JSON and writes an SSE event
// with the sequence number as the id field. Marshal failures are skipped.
// Wire-format serialization is delegated to [sse.WriteEvent] (ADR-0097).
func writeReplaySSEEvent[V any](w http.ResponseWriter, item SeqValue[V]) error {
	data, err := json.Marshal(item.Value)
	if err != nil {
		return nil //nolint:nilerr // skip unmarshalable, keep stream alive
	}

	return sse.WriteEvent(w, sse.Event{
		ID:   sse.NewEventID(strconv.FormatUint(item.Seq, 10)),
		Data: string(data),
	})
}

// replayMissedEvents writes events from the journal that the client missed
// (seq > afterSeq) and returns a dedup ring populated with the sequence
// numbers that were sent, so the live phase can skip overlapping events.
func replayMissedEvents[V any](
	w http.ResponseWriter,
	r *http.Request,
	replay *SSEReplay[V],
	cfg SSEConfig,
) (*dedup.Ring, error) {
	var afterSeq uint64

	if lastID := r.Header.Get("Last-Event-ID"); lastID != "" {
		if parsed, err := strconv.ParseUint(lastID, 10, 64); err == nil {
			afterSeq = parsed
		}
	}

	replayed := replay.Replay(afterSeq)

	// If capped, keep only the MOST RECENT events — the client cares about
	// current state more than historical completeness.
	if cfg.ReplayLimit > 0 && len(replayed) > cfg.ReplayLimit {
		replayed = replayed[len(replayed)-cfg.ReplayLimit:]
	}

	ring := dedup.NewRing(dedup.DefaultCapacity)

	for _, sv := range replayed {
		data, err := json.Marshal(sv.Value)
		if err != nil {
			continue
		}

		if err := sse.WriteEvent(w, sse.Event{
			ID:   sse.NewEventID(strconv.FormatUint(sv.Seq, 10)),
			Data: string(data),
		}); err != nil {
			return ring, err //nolint:wrapcheck
		}

		ring.Add(strconv.FormatUint(sv.Seq, 10))
	}

	return ring, nil
}

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
	return json.Marshal(s.Collections()) //nolint:wrapcheck
}
