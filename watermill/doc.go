// Package watermill provides protocol adapters between go-cqrs-lite event interfaces
// and the Watermill message bus library (github.com/ThreeDotsLabs/watermill).
//
// It translates between Watermill's message.Message and go-cqrs-lite's event.Event,
// enabling integration with Watermill's pub/sub infrastructure (Kafka, RabbitMQ, NATS, etc.).
//
// # Publisher Adapter
//
// Wraps an event.Publisher as a Watermill publisher:
//
//	adapter := watermill.NewPublisherAdapter(eventBus)
//	watermillPublisher := adapter // implements watermill.Publisher
//
// # Subscriber Adapter
//
// Subscribes to a go-cqrs-lite event.Bus and delivers events as Watermill messages:
//
//	adapter := watermill.NewSubscriberAdapter(eventBus)
//	messages, _ := adapter.Subscribe(ctx, "user.created")
//
// # Router Integration
//
// Watermill's Router connects publishers and subscribers with middleware
// (correlation IDs, retry, poison queue). Use the adapter functions to bridge
// go-cqrs-lite events into a Watermill router:
//
//	router, _ := message.NewRouter(message.RouterConfig{}, logger)
//
//	// Add middleware: correlation ID propagation + retry with backoff
//	router.AddMiddleware(watermill.CorrelationIDMiddleware())
//	router.AddMiddleware(watermill.NewRetryMiddleware(watermill.DefaultRetryConfig()))
//
//	// Bridge event bus to Watermill messages
//	subAdapter := watermill.NewSubscriberAdapter(eventBus)
//	pubAdapter := watermill.NewPublisherAdapter(eventBus)
//
//	router.AddHandler("project-events", "user.created", subAdapter,
//	    "projection-updated", pubAdapter, func(msg *message.Message) ([]*message.Message, error) {
//	        // process event, return new messages to publish
//	        return []*message.Message{}, nil
//	    })
//
//	router.Run(ctx) // blocks until context cancelled
//
// # Middleware Wrappers
//
// Pre-configured middleware wrappers simplify common patterns:
//
//	// Correlation ID propagation across handler chains
//	corrMiddleware := watermill.CorrelationIDMiddleware()
//
//	// Retry with exponential backoff (5 retries, 100ms→10s range)
//	retryMiddleware := watermill.NewRetryMiddleware(watermill.DefaultRetryConfig())
//
//	// Custom retry configuration
//	retryMiddleware = watermill.NewRetryMiddleware(watermill.RetryConfig{
//	    MaxRetries:      10,
//	    InitialInterval: 50 * time.Millisecond,
//	    MaxInterval:     5 * time.Second,
//	    Multiplier:      1.5,
//	})
//
// # Command Bridge
//
// The same adapter pattern bridges go-cqrs-lite command.Bus to Watermill,
// enabling command distribution across processes via Kafka, NATS, Redis, etc.
//
//	// Full command.Bus backed by Watermill GoChannel (single-process)
//	bus := watermill.NewCommandBus()
//	defer bus.Close()
//	bus.Subscribe("user.create", handlerFunc)
//	bus.Publish(ctx, cmd)
//
//	// Multi-process: inject a broker backend
//	bus := watermill.NewCommandBus(
//	    watermill.WithCommandBackend(natsPublisher, natsSubscriber, closer),
//	)
//
//	// Or wrap an existing message.Publisher as command.Publisher
//	pub := watermill.NewCommandPublisher(wmPublisher, "commands")
//
// Commands carry identity (type, aggregate ID) and tracing metadata. Payload
// data is encoded via custom metadata (same pattern as transport/grpc). Use
// [CommandToMessage] and [MessageToCommand] for the wire protocol.
//
// # Broker Backends (NATS, Redis, Kafka)
//
// The bridge works with any Watermill-compatible broker plugin. Inject the
// publisher+subscriber pair via WithCommandBackend (and/or WithBackend for
// events). No additional transport modules are needed.
//
//	// NATS JetStream (requires watermill-nats plugin)
//	js, _ := jetstream.New(natsURL, natsCtx, logger)
//	bus := watermill.NewCommandBus(
//	    watermill.WithCommandBackend(js, js, js),
//	)
//	busEB := watermill.NewEventBus(
//	    watermill.WithBackend(js, js, js),
//	)
//
//	// Redis Streams (requires watermill-redis-stream plugin)
//	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	pub, _ := redisStream.NewPublisher(redisStream.PublisherConfig{Client: rc}, logger)
//	sub, _ := redisStream.NewSubscriber(redisStream.SubscriberConfig{Client: rc}, logger)
//	bus := watermill.NewCommandBus(
//	    watermill.WithCommandBackend(pub, sub, io.CloserFunc(func() error { return rc.Close() })),
//	)
package watermill
