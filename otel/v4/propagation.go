package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
)

const correlationIDKey = "cqrs.correlation_id"

// WithCorrelationID stores a correlation ID in the context via OTel baggage.
// When propagators are configured, the correlation ID travels across service
// boundaries in W3C baggage headers — enabling distributed trace correlation
// without modifying every function signature.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	member, err := baggage.NewMember(correlationIDKey, id)
	if err != nil {
		return ctx
	}

	bag, err := baggage.New(member)
	if err != nil {
		return ctx
	}

	return baggage.ContextWithBaggage(ctx, bag)
}

// CorrelationIDFromContext retrieves the correlation ID from the context's
// OTel baggage. Returns empty string if not set.
func CorrelationIDFromContext(ctx context.Context) string {
	bag := baggage.FromContext(ctx)
	member := bag.Member(correlationIDKey)

	return member.Value()
}

// NewTextMapPropagator returns a composite propagator combining W3C trace
// context and W3C baggage propagation. This is the recommended setup for
// CQRS systems — trace context propagates spans, baggage propagates
// correlation IDs across service boundaries.
//
//	otel.SetTextMapPropagator(cqrsotel.NewTextMapPropagator())
func NewTextMapPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// TextMapPropagator returns the globally configured propagator. This is a
// re-export of otel.GetTextMapPropagator so downstream modules can inject and
// extract W3C trace context without a direct go.opentelemetry.io/otel
// dependency (design principle: OTel through otel/).
func TextMapPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}
