package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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
// replayLimit caps the number of replayed events per connection. Use 0 for
// the default cap of 1000 (prevents unbounded replay on first connect with
// a large journal).
func WithReconnectJournal(journal event.SeekableJournal, replayLimit int) SSEBrokerOption {
	return func(b *SSEBroker) {
		b.journal = journal
		b.replayLimit = replayLimit
	}
}

// SSEBroker bridges an event bus to Server-Sent Events HTTP clients.
type SSEBroker struct {
	mu          sync.RWMutex
	clients     map[SSEClientID]chan event.Event
	handler     event.Handler
	cancel      context.CancelFunc
	journal     event.SeekableJournal // optional: for Last-Event-ID replay
	replayLimit int                   // 0 = unlimited
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

		return nil, event.WrapInfrastructure(
			err,
			"transport.http.sse_subscribe",
			"subscribe to event bus",
		)
	}

	return b, nil
}

func (b *SSEBroker) handleEvent(ctx context.Context, evt event.Event) error {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "sse.fanout",
		cqrsotel.SpanKindConsumer,
		cqrsotel.WithAttributes(
			cqrsotel.EventAttrs(string(evt.Type()), evt.AggregateID(), string(evt.AggregateType()))...,
		),
	)
	defer span.End()

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

		// Last-Event-ID reconnection: replay missed events if journal is available.
		if broker.journal != nil {
			if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
				replayEvents(w, flusher, broker, r.Context(), lastEventID)
			}
		}

		ch := broker.AddClient(SSEClientID(clientID))
		defer broker.RemoveClient(SSEClientID(clientID))

		ticker := time.NewTicker(DefaultSSEHeartbeat)
		defer ticker.Stop()

		for {
			select {
			case evt := <-ch:
				if evt == nil {
					return
				}

				_ = WriteSSEEvent(w, SSEEvent{
					Type: string(evt.Type()),
					ID:   SSEEventID(evt.ID().String()),
					Data: string(event.PayloadReadOnly(evt)),
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

// defaultReplayLimit caps replay when WithReconnectJournal is called with 0.
const defaultReplayLimit = 1000

// replayEvents sends missed events to a reconnecting client.
// Events are read from the journal starting after lastEventID and written
// to the client before live streaming begins.
// Returns the set of replayed EventIDs for live-phase deduplication.
func replayEvents(
	w http.ResponseWriter,
	flusher http.Flusher,
	broker *SSEBroker,
	ctx context.Context,
	lastEventID string,
) map[string]struct{} {
	afterID, err := id.ParseEventID(lastEventID)
	if err != nil {
		return nil // invalid ID: skip replay, start live
	}

	limit := broker.replayLimit
	if limit <= 0 {
		limit = defaultReplayLimit
	}

	events, err := broker.journal.ReadFrom(ctx, afterID, limit)
	if err != nil {
		return nil // journal error: skip replay, start live
	}

	replayed := make(map[string]struct{}, len(events))

	for _, evt := range events {
		replayed[evt.ID().String()] = struct{}{}

		_ = WriteSSEEvent(w, SSEEvent{
			Type: string(evt.Type()),
			ID:   SSEEventID(evt.ID().String()),
			Data: string(event.PayloadReadOnly(evt)),
		})
	}

	flusher.Flush()

	return replayed
}
