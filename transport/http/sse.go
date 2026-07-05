package http

import (
	"context"
	"net/http"
	"sync"
	"time"

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

// sseReplayBatchSize is the batch size for unlimited streaming replay.
// Each batch is fetched from the journal, written to the client, and flushed
// before the next batch is loaded — keeping memory bounded.
const sseReplayBatchSize = 500

// sseDedupRingCapacity is the maximum number of event IDs retained for
// replay→live deduplication. Only the tail of the replay stream can overlap
// with the live channel (events published during the replay window), and the
// live channel buffer is bounded at sseChannelBufSize. A ring of 1024 entries
// gives a 10x safety margin while bounding memory to ~90KB regardless of
// journal size.
const sseDedupRingCapacity = 1024

// SSEBroker bridges an event bus to Server-Sent Events HTTP clients.
type SSEBroker struct {
	mu            sync.RWMutex
	clients       map[SSEClientID]chan event.Event
	handler       event.Handler
	cancel        context.CancelFunc
	journal       event.SeekableJournal // optional: for Last-Event-ID replay
	replayLimit   int                   // <=0 = unlimited (batch streaming); >0 = bounded
	replayTimeout time.Duration         // <=0 = no limit; >0 = max replay duration
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

		// Register the client BEFORE replay so concurrent live events are
		// buffered in the channel rather than lost during the replay window.
		ch := broker.AddClient(SSEClientID(clientID))
		defer broker.RemoveClient(SSEClientID(clientID))

		// Last-Event-ID reconnection: replay missed events if journal is available.
		// The dedup ring is carried into the live loop to suppress duplicates.
		replayed := (*dedupRing)(nil)

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
