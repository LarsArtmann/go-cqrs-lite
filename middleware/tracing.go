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

// tracedHandler constrains H to types with a Type() string method.
type tracedHandler interface {
	~struct{}
	Type() string
}

// withTracing wraps a handler with OpenTelemetry tracing.
func withTracing[H tracedHandler](
	tracer trace.Tracer,
	spanName string,
	spanKind trace.SpanKind,
	msgKind string,
	next func(context.Context, H) error,
) func(context.Context, H) error {
	return func(ctx context.Context, h H) error {
		attrTypeKey := "cqrs." + msgKind + ".type"

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(spanKind),
			trace.WithAttributes(
				attribute.String("cqrs.message.kind", msgKind),
				attribute.String(attrTypeKey, h.Type()),
			),
		)
		defer span.End()

		err := next(ctx, h)
		if err != nil {
			recordError(span, err)
		}

		return err
	}
}

// CommandTracing creates an OpenTelemetry span for each command handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := otel.GetTracerProvider().Tracer(instrumentationName)
//	mw := middleware.CommandTracing(tracer)
func CommandTracing(tracer trace.Tracer) command.Middleware {
	return func(next command.Handler) command.Handler {
		return withTracing(tracer, "command.handle", trace.SpanKindServer, "command", next)
	}
}

// EventTracing creates an OpenTelemetry span for each event handled.
// The tracer is typically obtained from a trace.TracerProvider:
//
//	tracer := otel.GetTracerProvider().Tracer(instrumentationName)
//	mw := middleware.EventTracing(tracer)
func EventTracing(tracer trace.Tracer) event.Middleware {
	return func(next event.Handler) event.Handler {
		return withTracing(tracer, "event.handle", trace.SpanKindConsumer, "event", next)
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
					attribute.String("cqrs.query.type", qry.Type()),
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
