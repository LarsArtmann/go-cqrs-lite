// Package http provided HTTP transport adapters for CQRS event streams:
// SSEBroker bridged an event.Bus to Server-Sent Events HTTP clients. The
// library doctrine is "not a framework — no opinionated transport": generic
// delivery mechanisms belong in dedicated libraries, and better ones now
// exist:
//
//   - SSE delivery: github.com/larsartmann/go-sse — the standalone SSE
//     library already used by metaengine.ServeSSE for read-model push.
//   - Broker transport: the watermill/ module bridges event.Bus and
//     command.Bus to any Watermill-compatible broker (NATS, Redis, Kafka)
//     via NewEventPublisher / WithBackend.
//   - HTTP UI delivery: cqrs-htmx.
//
// # DESIGN DECISIONS — Why things are the way they are
//
// Fanout: Sequential, non-blocking send (not a worker pool)
//
//	handleEvent iterates clients under RLock with a non-blocking channel send
//	(select + default). This is correct for the typical deployment (<500
//	clients). A worker pool was prototyped and reverted: it reinvented a
//	message router inside the broker, added ~130 lines, and the non-blocking
//	send means no slow client can ever block the broker goroutine or other
//	clients. Consumers needing massive fanout should use a message broker
//	(Watermill + NATS/Redis) or load-balanced SSE endpoints, not a single
//	process's in-memory fanout.
//
// Backpressure: Drop-newest (not drop-oldest)
//
//	When a client channel buffer (100 events) is full, the incoming event is
//	dropped. This preserves FIFO order for already-buffered events and avoids
//	the complexity of drain-and-resend. A drop-oldest policy was prototyped
//	and reverted: it adds a race window (drain → resend) for marginal benefit.
//	Consumers needing strict delivery guarantees should use CatchUpSubscriber
//	(which has checkpointing + DLQ) instead of raw SSE.
//
// WebSocket: Not included (YAGNI for a library)
//
//	SSE is unidirectional (server→client) and sufficient for event streaming.
//	WebSocket adds bidirectional complexity, connection upgrade negotiation,
//	and a completely different lifecycle model. Consumers who need
//	bidirectional transport (command + event over one connection) should use
//	transport/grpc (itself deprecated, removed in v5 — use the watermill/
//	bridge instead), which provided both CommandClient and QueryClient.
//
// Compression: Not included (proxy-level concern)
//
//	SSE compression (gzip Content-Encoding) is better handled by the reverse
//	proxy (Nginx, Cloudflare, ALB) which already compresses text/event-stream
//	responses. Adding gzip in the library would double-compress when a proxy
//	is present and add CPU overhead for each connection.
//
// Deprecated: This module is deprecated per ADR-0127 and will be removed in
// v5. Use github.com/larsartmann/go-sse for SSE delivery or the watermill/
// bridge for broker transport. New projects must not import this module.
package http
