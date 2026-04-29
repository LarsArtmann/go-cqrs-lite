package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/larsartmann/go-cqrs-lite/middleware"

// tracerProvider allows overriding the global tracer provider for testing.
//
//nolint:gochecknoglobals // package-level tracer override for testing
var tracerProvider = otel.GetTracerProvider()

// SetTracerProvider overrides the tracer provider used by all tracing middleware.
// Call before constructing dispatchers. Defaults to otel.GetTracerProvider().
func SetTracerProvider(tp trace.TracerProvider) {
	tracerProvider = tp
}

// CommandTracing creates an OpenTelemetry span for each command handled.
func CommandTracing() command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			tr := tracerProvider.Tracer(instrumentationName)

			ctx, span := tr.Start(ctx, "command.handle",
				trace.WithAttributes(
					attribute.String("cqrs.message.kind", "command"),
					attribute.String("cqrs.command.type", string(cmd.Type())),
				),
			)
			defer span.End()

			err := next(ctx, cmd)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			return err
		}
	}
}

// EventTracing creates an OpenTelemetry span for each event handled.
func EventTracing() event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			tr := tracerProvider.Tracer(instrumentationName)

			ctx, span := tr.Start(ctx, "event.handle",
				trace.WithAttributes(
					attribute.String("cqrs.message.kind", "event"),
					attribute.String("cqrs.event.type", string(evt.Type())),
				),
			)
			defer span.End()

			err := next(ctx, evt)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			return err
		}
	}
}

// QueryTracing creates an OpenTelemetry span for each query handled.
func QueryTracing() query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, qry query.Query) (any, error) {
			tr := tracerProvider.Tracer(instrumentationName)

			ctx, span := tr.Start(ctx, "query.handle",
				trace.WithAttributes(
					attribute.String("cqrs.message.kind", "query"),
					attribute.String("cqrs.query.type", string(qry.Type())),
				),
			)
			defer span.End()

			result, err := next(ctx, qry)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			return result, err
		}
	}
}
