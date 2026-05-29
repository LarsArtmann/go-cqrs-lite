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

// CommandTracing creates an OpenTelemetry span for each command handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.CommandTracing(tracer)
func CommandTracing(tracer trace.Tracer) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			ctx, span := tracer.Start(
				ctx, "command.handle",
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(cqrsotel.CommandAttrs(
					string(cmd.Type()),
					cmd.AggregateID(),
				)...),
			)
			defer span.End()

			err := next(ctx, cmd)
			if err != nil {
				cqrsotel.RecordError(span, err)
			}

			return err
		}
	}
}

// EventTracing creates an OpenTelemetry span for each event handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.EventTracing(tracer)
func EventTracing(tracer trace.Tracer) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			ctx, span := tracer.Start(
				ctx, "event.handle",
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(cqrsotel.EventAttrs(
					string(evt.Type()),
					evt.AggregateID(),
					string(evt.AggregateType()),
				)...),
			)
			defer span.End()

			span.SetAttributes(
				attribute.Int(cqrsotel.AttrAggregateVersion, int(evt.Version())),
			)

			err := next(ctx, evt)
			if err != nil {
				cqrsotel.RecordError(span, err)
			}

			return err
		}
	}
}

// QueryTracing creates an OpenTelemetry span for each query handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.QueryTracing(tracer)
func QueryTracing(tracer trace.Tracer) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, qry query.Query) (any, error) {
			ctx, span := tracer.Start(
				ctx, "query.handle",
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(cqrsotel.QueryAttrs(
					string(qry.Type()),
				)...),
			)
			defer span.End()

			result, err := next(ctx, qry)
			if err != nil {
				cqrsotel.RecordError(span, err)
			}

			return result, err
		}
	}
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

			return err
		})
	}
}
