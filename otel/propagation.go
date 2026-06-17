package otel

import (
	"context"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
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

// NewCQRSViews returns SDK metric views optimized for CQRS operations.
// The views configure histogram boundaries covering the typical CQRS
// operation latency range (0.05ms to 10s) and are applied to all
// instruments matching the "cqrs." prefix.
//
//	provider, _ := sdkmetric.NewMeterProvider(
//	    sdkmetric.WithReader(reader),
//	    sdkmetric.WithView(cqrsotel.NewCQRSViews()...),
//	)
func NewCQRSViews() []sdkmetric.View {
	return []sdkmetric.View{
		sdkmetric.NewView(
			sdkmetric.Instrument{ //nolint:exhaustruct // only Name is a filter criteria
				Name: "cqrs.",
			},
			sdkmetric.Stream{ //nolint:exhaustruct // only Aggregation is configured
				Aggregation: sdkmetric.AggregationExplicitBucketHistogram{ //nolint:exhaustruct // NoMinMax defaults to false
					Boundaries: CQRSHistogramBoundaries,
				},
			},
		),
	}
}
