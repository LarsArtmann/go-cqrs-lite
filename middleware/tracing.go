package middleware

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	"github.com/larsartmann/go-cqrs-lite/query"
)

// NewTracing creates a generic OpenTelemetry span for each message handled.
func NewTracing[M any](
	tracer trace.Tracer,
	spanName string,
	kind trace.SpanKind,
	attrs func(M) []attribute.KeyValue,
) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			ctx, span := tracer.Start(
				ctx, spanName,
				trace.WithSpanKind(kind),
				trace.WithAttributes(attrs(msg)...),
			)
			defer span.End()

			err := next(ctx, msg)
			if err != nil {
				cqrsotel.RecordError(span, err)
			}

			return err
		}
	}
}

// CommandTracing creates an OpenTelemetry span for each command handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.CommandTracing(tracer)
func CommandTracing(tracer trace.Tracer) command.Middleware {
	return AsCommand(
		NewTracing(
			tracer,
			"command.handle",
			trace.SpanKindServer,
			func(cmd command.Command) []attribute.KeyValue {
				return cqrsotel.CommandAttrs(string(cmd.Type()), cmd.AggregateID())
			},
		),
	)
}

// EventTracing creates an OpenTelemetry span for each event handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.EventTracing(tracer)
func EventTracing(tracer trace.Tracer) event.Middleware {
	return AsEvent(
		NewTracing(
			tracer,
			"event.handle",
			trace.SpanKindConsumer,
			func(evt event.Event) []attribute.KeyValue {
				attrs := cqrsotel.EventAttrs(
					string(evt.Type()),
					evt.AggregateID(),
					string(evt.AggregateType()),
				)

				return append(
					attrs,
					attribute.Int(cqrsotel.AttrAggregateVersion, int(evt.Version())),
				)
			},
		),
	)
}

// QueryTracing creates an OpenTelemetry span for each query handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.QueryTracing(tracer)
func QueryTracing(tracer trace.Tracer) query.Middleware {
	return AsQuery(
		NewTracing(
			tracer,
			"query.handle",
			trace.SpanKindServer,
			func(q query.Query) []attribute.KeyValue {
				return cqrsotel.QueryAttrs(string(q.Type()))
			},
		),
	)
}

// EventPublishTracing creates an OpenTelemetry span for each event publish operation.
// This wraps the Publish path on the event bus, creating a Producer span with
// attributes for the batch of events being published.
//
//	tracer := cqrsotel.NewTracer("middleware")
//	bus.UsePublish(middleware.EventPublishTracing(tracer))
func EventPublishTracing(tracer trace.Tracer) event.PublishMiddleware {
	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			attrs := []attribute.KeyValue{
				attribute.Int(cqrsotel.AttrEventCount, len(events)),
			}

			if len(events) > 0 {
				attrs = append(attrs, cqrsotel.EventAttrs(
					string(events[0].Type()),
					events[0].AggregateID(),
					string(events[0].AggregateType()),
				)...)
			}

			ctx, span := tracer.Start(
				ctx, "event.publish",
				trace.WithSpanKind(trace.SpanKindProducer),
				trace.WithAttributes(attrs...),
			)
			defer span.End()

			err := next.Publish(ctx, events...)
			if err != nil {
				cqrsotel.RecordError(span, err)
			}

			return err //nolint:wrapcheck // transparent proxy, caller wraps
		})
	}
}
