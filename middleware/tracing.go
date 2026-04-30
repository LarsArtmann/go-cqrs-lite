package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/larsartmann/go-cqrs-lite/middleware"

// recordError records an error on the span and sets error status.
func recordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// CommandTracing creates an OpenTelemetry span for each command handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := otel.GetTracerProvider().Tracer(instrumentationName)
//	mw := middleware.CommandTracing(tracer)
func CommandTracing(tracer trace.Tracer) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			ctx, span := tracer.Start(ctx, "command.handle",
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("cqrs.message.kind", "command"),
					attribute.String("cqrs.command.type", string(cmd.Type())),
				),
			)
			defer span.End()

			err := next(ctx, cmd)
			if err != nil {
				recordError(span, err)
			}

			return err
		}
	}
}

// EventTracing creates an OpenTelemetry span for each event handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := otel.GetTracerProvider().Tracer(instrumentationName)
//	mw := middleware.EventTracing(tracer)
func EventTracing(tracer trace.Tracer) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			ctx, span := tracer.Start(ctx, "event.handle",
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					attribute.String("cqrs.message.kind", "event"),
					attribute.String("cqrs.event.type", string(evt.Type())),
				),
			)
			defer span.End()

			err := next(ctx, evt)
			if err != nil {
				recordError(span, err)
			}

			return err
		}
	}
}

// QueryTracing creates an OpenTelemetry span for each query handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := otel.GetTracerProvider().Tracer(instrumentationName)
//	mw := middleware.QueryTracing(tracer)
func QueryTracing(tracer trace.Tracer) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, qry query.Query) (any, error) {
			ctx, span := tracer.Start(ctx, "query.handle",
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("cqrs.message.kind", "query"),
					attribute.String("cqrs.query.type", string(qry.Type())),
				),
			)
			defer span.End()

			result, err := next(ctx, qry)
			if err != nil {
				recordError(span, err)
			}

			return result, err
		}
	}
}
