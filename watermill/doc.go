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
package watermill
