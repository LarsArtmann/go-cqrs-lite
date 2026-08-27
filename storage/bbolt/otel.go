package bbolt

import (
	"context"
	"errors"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

const bboltComponent = "bbolt"

func tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(bboltComponent)
}

func startStreamSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			append(cqrsotel.StreamAttrs(ref.Type, ref.ID), extraAttrs...)...,
		),
	)
}

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

func startReadSpan(ctx context.Context, name string) cqrsotel.Span {
	_, span := cqrsotel.StartSpan(ctx, tracer(), name, cqrsotel.SpanKindClient)
	return span
}

func startLimitSpan(ctx context.Context, spanName string, limit int) cqrsotel.Span {
	_, span := cqrsotel.StartSpan(ctx, tracer(), spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("limit", limit)))
	return span
}

func finalizeScan[T any](
	span cqrsotel.Span,
	itemsOut []T,
	err error,
	code, msg, countAttr string,
) ([]T, error) {
	if err != nil { //art-dupl:accept cross-module OTel error recording — separate go.mod
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.Wrap(err, familyOrInfrastructure(err), code, msg)
	}

	span.SetAttributes(cqrsotel.AttrInt(countAttr, len(itemsOut)))
	return itemsOut, nil
}

// familyOrInfrastructure preserves an already-classified inner family (a
// Corruption from a decode failure must stay Corruption, not surface as
// Infrastructure) and defaults only unclassified errors to Infrastructure.
func familyOrInfrastructure(err error) errorfamily.Family {
	if _, ok := errors.AsType[errorfamily.Classified](err); ok {
		return errorfamily.Classify(err)
	}

	return errorfamily.Infrastructure
}

func reportScanErr(span cqrsotel.Span, err error, code, msg string) error {
	cqrsotel.RecordError(span, err)
	return errorfamily.Wrap(err, familyOrInfrastructure(err), code, msg)
}
