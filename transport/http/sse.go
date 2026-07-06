package http

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/larsartmann/go-cqrs-lite/dedup/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

// SSEClientID identifies a connected SSE client.
type SSEClientID string

func (c SSEClientID) String() string { return string(c) }

func (c SSEClientID) IsZero() bool { return c == "" }

// SSEBrokerOption configures an SSEBroker.
type SSEBrokerOption func(*SSEBroker)

// WithReconnectJournal enables Last-Event-ID reconnection.
// When a client sends the Last-Event-ID header, the broker replays missed
// events from the journal before starting live delivery.
// Events replayed from the journal are tracked by EventID and suppressed if
// they also arrive via the live bus (same dedup strategy as
// watermill.CatchUpSubscriber).
//
// replayLimit controls the maximum number of replayed events:
//   - replayLimit > 0: bounded replay capped at that many events.
//   - replayLimit <= 0: unlimited replay — events are streamed in batches
//     from the journal so memory stays bounded regardless of journal size.
//
// For a sensible bounded default, pass DefaultSSEReplayLimit (1000).
func WithReconnectJournal(journal event.SeekableJournal, replayLimit int) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.journal = journal
		b.replayLimit = replayLimit
	}
}

// DefaultSSEReplayLimit is the suggested bounded replay cap for callers who
// want a finite replay window. Pass it to WithReconnectJournal, or pass <= 0
// for unlimited streaming replay.
const DefaultSSEReplayLimit = 1000

// SSEReplayIncompleteEvent is the SSE event type sent when journal replay is
// cut short by a timeout (see WithReplayTimeout). The event carries no id
// field so it does not advance the client's Last-Event-ID. Clients receiving
// this event know they are behind and should reconnect with their latest
// received EventID (or use a backfill endpoint) to catch up.
const SSEReplayIncompleteEvent = "cqrs.replay.incomplete"

// WithReplayByteBudget caps unlimited replay by total payload bytes instead
// of event count. When the cumulative size of replayed event payloads exceeds
// the budget, replay stops and an SSEReplayIncompleteEvent advisory is sent.
//
// This is safer than count-based batching (sseReplayBatchSize) for journals
// containing very large payloads (e.g. 1MB+ blob events): a fixed count of 500
// such events would consume 500MB. The default budget
// (sseDefaultReplayByteBudget = 8MB) is a sensible bound; pass 0 to disable
// byte-budgeting (replay falls back to count-based batching).
//
// Applies only when replayLimit <= 0 (unlimited replay).
func WithReplayByteBudget(bytes int) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.replayByteBudget = bytes
	}
}

// WithReplayTimeout sets the maximum duration for journal replay before
// switching to live delivery. If replay is not complete when the timeout
// fires, the broker sends an advisory SSEReplayIncompleteEvent and begins
// live streaming.
//
// A timeout of zero (the default) means no limit — replay runs until the
// journal is exhausted. Use a non-zero timeout for browser-facing SSE where
// handler starvation must be avoided (e.g. a client reconnecting after a long
// offline period with a very large journal).
func WithReplayTimeout(d time.Duration) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.replayTimeout = d
	}
}

// WithReplayMetrics installs OpenTelemetry instruments for SSE replay
// observability (duration histogram, events counter, incomplete counter).
// Pass a *ReplayMetrics from NewReplayMetrics; nil disables metrics (no-op).
//
// Without this option, replay records only span attributes — useful in traces
// but invisible to dashboards. This option promotes replay telemetry to
// first-class OTel instruments scrapeable by Prometheus.
func WithReplayMetrics(metrics *ReplayMetrics) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.replayMetrics = metrics
	}
}

// WithDedupRingCapacity overrides the default SSE dedup ring capacity
// (sseDedupRingCapacity = 1024). The ring bounds replay→live deduplication
// memory at ~capacity × 90 bytes.
//
// Increase if your live channel buffer (sseChannelBufSize) is raised above
// the default 100. Decrease for memory-constrained deployments with small
// journals. Values <= 0 fall back to the default.
func WithDedupRingCapacity(capacity int) SSEBrokerOption {
	return func(b *SSEBroker) {
		if capacity <= 0 {
			capacity = sseDedupRingCapacity
		}

		b.dedupRingCap = capacity
	}
}

// sseReplayBatchSize is the default batch size for unlimited streaming replay.
// Each batch is fetched from the journal, written to the client, and flushed
// before the next batch is loaded — keeping memory bounded. For very large
// payloads (1MB+), prefer WithReplayByteBudget which bounds by total bytes
// rather than event count.
const sseReplayBatchSize = 500

// sseDefaultReplayByteBudget is the default byte budget for replay when
// WithReplayByteBudget is used. 8MB accommodates ~5000 typical 1.5KB events
// while keeping per-client memory bounded. A budget of 0 disables byte-budget
// mode (count-based sseReplayBatchSize is used instead).
const sseDefaultReplayByteBudget = 8 * 1024 * 1024

// sseDedupRingCapacity is the maximum number of event IDs retained for
// replay→live deduplication. Only the tail of the replay stream can overlap
// with the live channel (events published during the replay window), and the
// live channel buffer is bounded at sseChannelBufSize. A ring of 1024 entries
// gives a 10x safety margin while bounding memory to ~90KB regardless of
// journal size.
const sseDedupRingCapacity = 1024

// fanoutPolicy controls how handleEvent delivers an event to client channels.
type fanoutPolicy int

const (
	// fanoutSequential iterates clients under the read lock on the broker
	// goroutine. Best for typical deployments (<500 clients).
	fanoutSequential fanoutPolicy = iota
	// fanoutParallel dispatches to a worker pool so client channels don't
	// block each other. Use WithParallelFanout for high client counts.
	fanoutParallel
)

// dropPolicy controls what happens when a client channel is full.
type dropPolicy int

const (
	// dropNewest (default) drops the incoming event when the channel is full.
	// Non-blocking; preserves events already buffered (FIFO order honored
	// for buffered events).
	dropNewest dropPolicy = iota
	// dropOldest evicts the oldest buffered event to make room for the newest.
	// Use when consumers care about CURRENT state more than history (e.g.
	// dashboards showing the latest value). Slower than dropNewest under
	// sustained pressure because each eviction requires a drain + resend.
	dropOldest
)

// SSEBroker bridges an event bus to Server-Sent Events HTTP clients.
type SSEBroker struct {
	mu               sync.RWMutex
	clients          map[SSEClientID]*sseClient
	handler          event.Handler
	cancel           context.CancelFunc
	journal          event.SeekableJournal // optional: for Last-Event-ID replay
	replayLimit      int                   // <=0 = unlimited (batch streaming); >0 = bounded
	replayTimeout    time.Duration         // <=0 = no limit; >0 = max replay duration
	replayMetrics    *ReplayMetrics        // optional: OTel instruments for replay
	replayByteBudget int                   // <=0 = disabled; >0 = max bytes before stopping
	dedupRingCap     int                   // <=0 = default (sseDedupRingCapacity)
	fanout           fanoutPolicy
	drop             dropPolicy
	fanoutWorkers    int
}

// sseClient wraps a client channel with per-client accounting (dropped events,
// buffered depth) for Stats() observability.
type sseClient struct {
	ch      chan event.Event
	dropped atomic.Int64
}

// WithParallelFanout switches handleEvent from sequential per-client iteration
// to a worker pool of the given size. Each worker drains a dispatch queue;
// slow clients don't block the broker goroutine or other clients.
//
// workers <= 0 falls back to sequential (the default). Recommended: set
// workers to roughly the expected client count divided by 50 (e.g. 4 workers
// for ~200 clients). The pool is created lazily on first handleEvent.
func WithParallelFanout(workers int) SSEBrokerOption {
	return func(b *SSEBroker) {
		if workers > 0 {
			b.fanout = fanoutParallel
			b.fanoutWorkers = workers
		}
	}
}

// WithDropOldestPolicy changes the per-client backpressure policy from
// dropNewest (default — drop the incoming event when full) to dropOldest
// (evict the oldest buffered event to admit the newest). Use when consumers
// prioritize current state over history.
func WithDropOldestPolicy() SSEBrokerOption {
	return func(b *SSEBroker) {
		b.drop = dropOldest
	}
}

// NewSSEBroker creates a new SSE broker that subscribes to the given bus.
// Returns an error if bus subscription fails.
// Pass WithReconnectJournal to enable Last-Event-ID reconnection.
func NewSSEBroker(bus event.Bus, opts ...SSEBrokerOption) (*SSEBroker, error) {
	if bus == nil {
		return nil, event.NewInfrastructure("transport.http.nil_bus", "event bus is required")
	}

	_, cancel := context.WithCancel(context.Background())

	b := &SSEBroker{
		clients: make(map[SSEClientID]*sseClient),
		cancel:  cancel,
		fanout:  fanoutSequential,
		drop:    dropNewest,
	}

	for _, opt := range opts {
		opt(b)
	}

	b.handler = func(c context.Context, evt event.Event) error {
		return b.handleEvent(c, evt)
	}

	err := bus.SubscribeAll(b.handler)
	if err != nil {
		cancel()

		return nil, event.WrapInfrastructure(
			err,
			"transport.http.sse_subscribe",
			"subscribe to event bus",
		)
	}

	return b, nil
}

func (b *SSEBroker) handleEvent(ctx context.Context, evt event.Event) error {
	_, span := cqrsotel.StartSpan(
		ctx, tracer(), "sse.fanout",
		cqrsotel.SpanKindConsumer,
		cqrsotel.WithAttributes(
			cqrsotel.EventAttrs(
				string(evt.Type()),
				evt.AggregateID(),
				string(evt.AggregateType()),
			)...,
		),
	)
	defer span.End()

	b.mu.RLock()
	span.SetAttributes(cqrsotel.AttrInt("cqrs.sse.client_count", len(b.clients)))

	if b.fanout == fanoutParallel {
		b.fanoutParallelLocked(span, evt)
	} else {
		b.fanoutSequentialLocked(span, evt)
	}

	b.mu.RUnlock()

	return nil
}

// fanoutSequentialLocked iterates clients under the read lock. Callers must
// hold b.mu (RLock). Non-blocking: full channels drop per the drop policy.
func (b *SSEBroker) fanoutSequentialLocked(span cqrsotel.Span, evt event.Event) {
	var dropped int64

	for _, c := range b.clients {
		if !b.sendToClient(c, evt) {
			dropped++
		}
	}

	if dropped > 0 {
		span.SetAttributes(cqrsotel.AttrInt("cqrs.sse.dropped_clients", int(dropped)))
	}
}

// fanoutParallelLocked dispatches via a worker pool. Callers must hold b.mu.
// The pool is created lazily and sized to b.fanoutWorkers.
func (b *SSEBroker) fanoutParallelLocked(span cqrsotel.Span, evt event.Event) {
	clients := make([]*sseClient, 0, len(b.clients))
	for _, c := range b.clients {
		clients = append(clients, c)
	}

	workers := min(b.fanoutWorkers, len(clients))

	if workers == 0 {
		return
	}

	var (
		wg      sync.WaitGroup
		dropped atomic.Int64
	)

	chunk := (len(clients) + workers - 1) / workers

	for w := range workers {
		start := w * chunk

		end := min(start+chunk, len(clients))

		if start >= end {
			break
		}

		wg.Add(1)

		go func(slice []*sseClient) {
			defer wg.Done()

			for _, c := range slice {
				if !b.sendToClient(c, evt) {
					dropped.Add(1)
				}
			}
		}(clients[start:end])
	}

	wg.Wait()

	if d := dropped.Load(); d > 0 {
		span.SetAttributes(cqrsotel.AttrInt("cqrs.sse.dropped_clients", int(d)))
	}
}

// sendToClient delivers evt to a single client using the broker's drop policy.
// Returns false if the event was dropped (channel full). Callers must hold b.mu.
func (b *SSEBroker) sendToClient(c *sseClient, evt event.Event) bool {
	switch b.drop {
	case dropOldest:
		select {
		case c.ch <- evt:
			return true
		default:
			// Channel full — evict the oldest buffered event to make room.
			select {
			case <-c.ch: // drain oldest
			default:
			}

			select {
			case c.ch <- evt:
				c.dropped.Add(1)

				return false // we dropped an event (the oldest)
			default:
				c.dropped.Add(1)

				return false
			}
		}
	default: // dropNewest
		select {
		case c.ch <- evt:
			return true
		default:
			c.dropped.Add(1)

			return false
		}
	}
}

const sseChannelBufSize = 100

// AddClient registers a new SSE client and returns its event channel.
func (b *SSEBroker) AddClient(id SSEClientID) chan event.Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	c := &sseClient{ch: make(chan event.Event, sseChannelBufSize)}
	b.clients[id] = c

	return c.ch
}

// RemoveClient unregisters an SSE client.
// The channel is not closed to avoid send-on-closed-channel races with
// concurrent handleEvent calls. The channel will be garbage-collected
// once the SSE handler releases its reference.
func (b *SSEBroker) RemoveClient(id SSEClientID) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.clients, id)
}

// ClientCount returns the number of connected clients.
func (b *SSEBroker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.clients)
}

// Close shuts down the broker and disconnects all clients.
func (b *SSEBroker) Close() {
	b.cancel()

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, c := range b.clients {
		close(c.ch)
	}

	b.clients = make(map[SSEClientID]*sseClient)
}

// DefaultSSEHeartbeat is the default interval for SSE keepalive comment frames.
// Most ALB/Nginx/Cloudflare proxies kill idle connections after 60s.
const DefaultSSEHeartbeat = 15 * time.Second

// SSEHandler returns an HTTP handler that streams events to a client.
// The clientID is extracted from the query parameter "client".
// A heartbeat comment frame is sent every DefaultSSEHeartbeat to prevent
// proxy/load-balancer idle timeouts.
func SSEHandler(broker *SSEBroker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("client")
		if clientID == "" {
			http.Error(w, "missing client ID", http.StatusBadRequest)

			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)

			return
		}

		flusher.Flush()

		// Register the client BEFORE replay so concurrent live events are
		// buffered in the channel rather than lost during the replay window.
		ch := broker.AddClient(SSEClientID(clientID))
		defer broker.RemoveClient(SSEClientID(clientID))

		// Last-Event-ID reconnection: replay missed events if journal is available.
		// The dedup ring is carried into the live loop to suppress duplicates.
		var replayed *dedup.Ring

		if broker.journal != nil {
			if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
				replayed = replayEvents(w, flusher, broker, r.Context(), lastEventID)
			}
		}

		ticker := time.NewTicker(DefaultSSEHeartbeat)
		defer ticker.Stop()

		for {
			select {
			case evt := <-ch:
				if evt == nil {
					return
				}

				// Suppress events already delivered during replay.
				if replayed.Has(evt.ID().String()) {
					continue
				}

				_ = WriteSSEEvent(w, SSEEvent{
					Event: string(evt.Type()),
					ID:    NewSSEEventID(evt.ID().String()),
					Data:  string(event.PayloadReadOnly(evt)),
				})

				flusher.Flush()
			case <-ticker.C:
				_ = WriteSSEHeartbeat(w)

				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
}
