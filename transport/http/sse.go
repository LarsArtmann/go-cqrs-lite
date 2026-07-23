package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// SSEClientID identifies a connected SSE client.
type SSEClientID string

func (c SSEClientID) String() string { return string(c) }

func (c SSEClientID) IsZero() bool { return c == "" }

// SSEBrokerOption configures an SSEBroker.
type SSEBrokerOption func(*SSEBroker)

// SSEBroker bridges an event bus to Server-Sent Events HTTP clients.
type SSEBroker struct {
	mu               sync.RWMutex
	clients          map[SSEClientID]chan event.Event
	handler          event.Handler
	cancel           context.CancelFunc
	journal          event.SeekableJournal    // optional: for Last-Event-ID replay
	replayLimit      int                      // <=0 = unlimited (batch streaming); >0 = bounded
	replayTimeout    time.Duration            // <=0 = no limit; >0 = max replay duration
	replayMetrics    *ReplayMetrics           // optional: OTel instruments for replay
	replayByteBudget int                      // 0 = auto-default (8MB for unlimited replay); -1 = explicitly disabled; >0 = explicit budget
	dedupRingCap     int                      // <=0 = default (sseDedupRingCapacity)
	retryInterval    time.Duration            // <=0 = default (DefaultSSERetryInterval)
	eventFilter      func(event.Type) bool    // nil = forward all events
	payloadTransform func(event.Event) []byte // nil = raw payload (backward compatible)
}

// payloadForWire returns the bytes to send to SSE clients. If a payload
// transform is configured, it is applied; otherwise the raw event payload
// bytes are returned (backward compatible).
func (b *SSEBroker) payloadForWire(evt event.Event) []byte {
	if b.payloadTransform != nil {
		return b.payloadTransform(evt)
	}

	return event.PayloadReadOnly(evt)
}

// NewSSEBroker creates a new SSE broker that subscribes to the given bus.
// Returns an error if bus subscription fails.
// Pass WithReconnectJournal to enable Last-Event-ID reconnection.
func NewSSEBroker(bus event.Bus, opts ...SSEBrokerOption) (*SSEBroker, error) {
	if bus == nil {
		return nil, errorfamily.NewInfrastructure("transport.http.nil_bus", "event bus is required")
	}

	_, cancel := context.WithCancel(context.Background())

	b := &SSEBroker{
		clients: make(map[SSEClientID]chan event.Event),
		cancel:  cancel,
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

		return nil, errorfamily.WrapInfrastructure(
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
				evt.StreamID(),
				string(evt.StreamType()),
			)...,
		),
	)
	defer span.End()

	if b.eventFilter != nil && !b.eventFilter(evt.Type()) {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	span.SetAttributes(cqrsotel.AttrInt("cqrs.sse.client_count", len(b.clients)))

	for _, ch := range b.clients {
		select {
		case ch <- evt:
		default:
		}
	}

	return nil
}

const sseChannelBufSize = 100

// AddClient registers a new SSE client and returns its event channel.
func (b *SSEBroker) AddClient(id SSEClientID) chan event.Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan event.Event, sseChannelBufSize)
	b.clients[id] = ch

	return ch
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

// Journal returns the seekable journal used for Last-Event-ID replay and
// REST backfill. Returns nil if no journal was configured via
// WithReconnectJournal. BackfillHandler checks for this and returns an error.
func (b *SSEBroker) Journal() event.SeekableJournal {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.journal
}

// PayloadTransform returns the configured payload transform function, or nil
// if none was set. Used by BackfillHandler to apply the same transform that
// SSE clients receive.
func (b *SSEBroker) PayloadTransform() func(event.Event) []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.payloadTransform
}

// Close shuts down the broker and disconnects all clients.
func (b *SSEBroker) Close() {
	b.cancel()

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.clients {
		close(ch)
	}

	b.clients = make(map[SSEClientID]chan event.Event)
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

		// Send retry interval so browsers reconnect predictably after drops.
		retry := broker.retryInterval
		if retry <= 0 {
			retry = DefaultSSERetryInterval
		}

		_ = WriteSSERetry(w, int(retry.Milliseconds()))

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
					Data:  string(broker.payloadForWire(evt)),
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
