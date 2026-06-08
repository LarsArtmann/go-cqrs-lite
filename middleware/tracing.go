package middleware

import (
	"context"


	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// NewTracing creates a generic OpenTelemetry span for each message handled.
func NewTracing[M any](
	tracer cqrsotel.Tracer,
	spanName string,
	kind cqrsotel.SpanKind,
	attrs func(M) []cqrsotel.KeyValue,
) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			ctx, span := tracer.Start(
				ctx, spanName,
				cqrsotel.WithSpanKind(kind),
				cqrsotel.WithAttributes(attrs(msg)...),
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
// The tracer is typically obtained from a cqrsotel.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.CommandTracing(tracer)
func CommandTracing(tracer cqrsotel.Tracer) command.Middleware {
	return AsCommand(
		NewTracing(
			tracer,
			"command.handle",
			cqrsotel.SpanKindServer,
			func(cmd command.Command) []cqrsotel.KeyValue {
				return cqrsotel.CommandAttrs(string(cmd.Type()), cmd.AggregateID())
			},
		),
	)
}

// EventTracing creates an OpenTelemetry span for each event handled.
// The tracer is typically obtained from a cqrsotel.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.EventTracing(tracer)
func EventTracing(tracer cqrsotel.Tracer) event.Middleware {
	return AsEvent(
		NewTracing(
			tracer,
			"event.handle",
			cqrsotel.SpanKindConsumer,
			func(evt event.Event) []cqrsotel.KeyValue {
				attrs := cqrsotel.EventAttrs(
					string(evt.Type()),
					evt.AggregateID(),
					string(evt.AggregateType()),
				)

				return append(
					attrs,
					cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, int(evt.Version())),
				)
			},
		),
	)
}

// QueryTracing creates an OpenTelemetry span for each query handled.
// The tracer is typically obtained from a cqrsotel.TracerProvider:
//
//	tracer := cqrsotel.NewTracer("middleware")
//	mw := middleware.QueryTracing(tracer)
func QueryTracing(tracer cqrsotel.Tracer) query.Middleware {
	return AsQuery(
		NewTracing(
			tracer,
			"query.handle",
			cqrsotel.SpanKindServer,
			func(q query.Query) []cqrsotel.KeyValue {
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
func EventPublishTracing(tracer cqrsotel.Tracer) event.PublishMiddleware {
	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			attrs := []cqrsotel.KeyValue{
				cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(events)),
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
				cqrsotel.WithSpanKind(cqrsotel.SpanKindProducer),
				cqrsotel.WithAttributes(attrs...),
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
