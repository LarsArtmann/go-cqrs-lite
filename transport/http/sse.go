package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// SSEClientID identifies a connected SSE client.
type SSEClientID string

func (c SSEClientID) String() string { return string(c) }

func (c SSEClientID) IsZero() bool { return c == "" }

// SSEBroker bridges an event bus to Server-Sent Events HTTP clients.
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[SSEClientID]chan event.Event
	handler event.Handler
	cancel  context.CancelFunc
}

// NewSSEBroker creates a new SSE broker that subscribes to the given bus.
// Returns an error if bus subscription fails.
func NewSSEBroker(bus event.Bus) (*SSEBroker, error) {
	if bus == nil {
		return nil, event.NewInfrastructure("transport.http.nil_bus", "event bus is required")
	}

	_, cancel := context.WithCancel(context.Background())

	b := &SSEBroker{ //nolint:exhaustruct // mu zero-value is ready, handler set below
		clients: make(map[SSEClientID]chan event.Event),
		cancel:  cancel,
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

func (b *SSEBroker) handleEvent(_ context.Context, evt event.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

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
