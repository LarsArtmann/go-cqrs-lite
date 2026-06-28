package watermill

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

// metadataCarrier adapts message.Metadata to the propagation.TextMapCarrier
// interface, enabling W3C trace context injection/extraction into Watermill
// message metadata. The traceparent and tracestate keys are standard W3C
// headers; they are ignored by MessageToEvent's field mapping and are
// harmless to consumers that do not extract them.
type metadataCarrier struct {
	md message.Metadata
}

func (c metadataCarrier) Get(key string) string {
	return c.md.Get(key)
}

func (c metadataCarrier) Set(key, value string) {
	c.md.Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for k := range c.md {
		keys = append(keys, k)
	}

	return keys
}

// injectTraceContext injects the W3C trace context from ctx into the message
// metadata using the globally configured propagator. When no propagator is
// set (the default no-op), this is a no-op. Call this after creating a
// producer span so the consumer span in another process links automatically.
func injectTraceContext(ctx context.Context, msg *message.Message) {
	cqrsotel.TextMapPropagator().Inject(ctx, metadataCarrier{md: msg.Metadata})
}

// ExtractContext returns ctx enriched with W3C trace context extracted from
// the message metadata. Consumers should call this at the start of their
// handler (or use TraceContextMiddleware) so any consumer span created in the
// handler links to the producer span that published the message.
//
// When no propagator is configured or no trace context is present, the
// original context is returned unchanged.
func ExtractContext(ctx context.Context, msg *message.Message) context.Context {
	return cqrsotel.TextMapPropagator().Extract(ctx, metadataCarrier{md: msg.Metadata})
}

// TraceContextMiddleware is a Watermill router middleware that extracts W3C
// trace context from incoming message metadata and sets it on the message
// context. Add it to your router so handler spans link to producer spans
// across process boundaries:
//
//	router.AddMiddleware(watermill.TraceContextMiddleware())
//
// This is the consumer-side counterpart to the automatic injection performed
// by EventPublisher and CommandPublisher on publish.
func TraceContextMiddleware() message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			ctx := ExtractContext(msg.Context(), msg)
			msg.SetContext(ctx)

			return h(msg)
		}
	}
}
