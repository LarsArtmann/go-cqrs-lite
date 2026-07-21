package pebble

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

const pebbleComponent = "pebble"

// tracer returns an OpenTelemetry tracer scoped to the pebble module.
// Uses the global TracerProvider, which returns a no-op tracer when no
// provider is configured.
func tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(pebbleComponent)
}

// startAggregateSpan creates a span for an aggregate-scoped snapshot operation
// with standard aggregate type and ID attributes.
func startAggregateSpan(
	ctx context.Context,
	spanName string,
	ref id.AggregateRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			append(cqrsotel.AggregateAttrs(ref.Type, ref.ID), extraAttrs...)...,
		),
	)
}

// startProjectionSpan creates a span for a checkpoint operation scoped to
// a projection name.
func startProjectionSpan(
	ctx context.Context,
	spanName, projectionName string,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrProjectionName, projectionName),
		),
	)
}

// idOrEmpty returns the string form of id, or "" when id is the zero value.
// Used to convert optional cursor IDs (CommandID, RequestID, EventID) into
// the skip-ID parameter for pebble scans.
func idOrEmpty[T interface {
	IsZero() bool
	String() string
}](id T) string {
	if id.IsZero() {
		return ""
	}

	return id.String()
}

// finalizeScan records err (if any) on span, otherwise stamps countAttr with
// the result count. Replaces the
//
//	if err != nil { RecordError; return nil, WrapInfrastructure }
//	span.SetAttributes(AttrInt(countAttr, len(items)))
//	return items, nil
//
// idiom that appears in every pebble read path. itemsOut is returned
// unchanged on success so callers can write `return finalizeScan(...)`.
func finalizeScan[T any](
	span cqrsotel.Span,
	itemsOut []T,
	err error,
	code, msg, countAttr string,
) ([]T, error) {
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, errorfamily.WrapInfrastructure(err, code, msg)
	}

	span.SetAttributes(cqrsotel.AttrInt(countAttr, len(itemsOut)))

	return itemsOut, nil
}

// reportScanErr records err on span and returns it wrapped as Infrastructure.
// Use for read paths that do not stamp a count attribute — replaces the
//
//	if err != nil { RecordError; return nil, WrapInfrastructure }
//
// idiom.
func reportScanErr(span cqrsotel.Span, err error, code, msg string) error {
	cqrsotel.RecordError(span, err)

	return errorfamily.WrapInfrastructure(err, code, msg)
}

// startLimitSpan opens a SpanKindClient span stamped with the "limit"
// attribute. Shared by the ReadFrom-style methods on EventStore (journal +
// stream), CommandStore, and QueryStore so the limit-attribute shape lives
// in one place.
func startLimitSpan(ctx context.Context, spanName string, limit int) cqrsotel.Span {
	_, span := cqrsotel.StartSpan(ctx, tracer(), spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("limit", limit)))

	return span
}
