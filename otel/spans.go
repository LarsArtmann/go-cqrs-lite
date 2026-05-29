package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan starts a new span with the given name and kind, returning the
// updated context and the span. The caller must call span.End().
func StartSpan(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	kind trace.SpanKind,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	allOpts := make([]trace.SpanStartOption, 0, len(opts)+1)
	allOpts = append(allOpts, trace.WithSpanKind(kind))
	allOpts = append(allOpts, opts...)

	return tracer.Start(ctx, name, allOpts...)
}

// RecordError records an error on the span and sets the span status to Error.
func RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// EndWithError ends the span, recording the error if non-nil.
func EndWithError(span trace.Span, err error) {
	if err != nil {
		RecordError(span, err)
	}

	span.End()
}

// SpanFromContext returns the current span from the context.
// This is a convenience wrapper around trace.SpanFromContext.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ComponentTracer returns a tracer name for a go-cqrs-lite component.
// Example: ComponentTracer("storage") → "github.com/larsartmann/go-cqrs-lite/storage"
func ComponentTracer(component string) string {
	return fmt.Sprintf("%s/%s", Name, component)
}
