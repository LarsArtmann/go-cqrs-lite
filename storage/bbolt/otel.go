package bbolt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

const component = "bbolt"

func tracer() cqrsotel.Tracer { return cqrsotel.NewTracer(component) }

func startStreamSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(ctx, tracer(), spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			append(cqrsotel.StreamAttrs(ref.Type, ref.ID), extraAttrs...)...,
		))
}

func startReadSpan(ctx context.Context, name string) cqrsotel.Span {
	_, span := cqrsotel.StartSpan(ctx, tracer(), name, cqrsotel.SpanKindClient)
	return span
}

func startProjectionSpan(
	ctx context.Context,
	spanName, projectionName string,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(ctx, tracer(), spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrProjectionName, projectionName),
		))
}
