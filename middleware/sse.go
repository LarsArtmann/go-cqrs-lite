package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// SSEBroker bridges an event bus to Server-Sent Events HTTP clients.
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[string]chan event.Event
	handler event.Handler
	cancel  context.CancelFunc
}

// NewSSEBroker creates a new SSE broker that subscribes to the given bus.
func NewSSEBroker(bus event.Bus) *SSEBroker {
	ctx, cancel := context.WithCancel(context.Background())

	b := &SSEBroker{
		clients: make(map[string]chan event.Event),
		cancel:  cancel,
	}

	b.handler = func(c context.Context, evt event.Event) error {
		return b.handleEvent(c, evt)
	}

	if err := bus.SubscribeAll(b.handler); err != nil {
		cancel()

		return nil
	}

	go func() {
		<-ctx.Done()
	}()

	return b
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

// AddClient registers a new SSE client and returns its event channel.
func (b *SSEBroker) AddClient(id string) chan event.Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan event.Event, 100)
	b.clients[id] = ch

	return ch
}

// RemoveClient unregisters an SSE client.
func (b *SSEBroker) RemoveClient(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.clients[id]; ok {
		close(ch)
		delete(b.clients, id)
	}
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

	b.clients = make(map[string]chan event.Event)
}

// SSEHandler returns an HTTP handler that streams events to a client.
// The clientID is extracted from the query parameter "client".
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

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)

			return
		}

		ch := broker.AddClient(clientID)
		defer broker.RemoveClient(clientID)

		for {
			select {
			case evt, ok := <-ch:
				if !ok {
					return
				}

				fmt.Fprintf(w, "event: %s\n", evt.Type())
				fmt.Fprintf(w, "id: %s\n", evt.ID().String())
				fmt.Fprintf(w, "data: %s\n\n", string(evt.Payload()))
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
}
