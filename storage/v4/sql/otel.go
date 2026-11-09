package sql

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

const storageComponent = "storage"

// Tracer returns an OpenTelemetry tracer for the storage module.
func Tracer() cqrsotel.Tracer {
	return cqrsotel.NewTracer(storageComponent)
}

// dbSystem maps a Dialect to its OpenTelemetry db.system semantic-convention
// value. DuckDB is not in the semconv registry, so it uses its lowercase
// product name. Custom dialects return "" — no attribute is stamped rather
// than a wrong one.
func dbSystem(d Dialect) string {
	switch d.(type) {
	case PostgresDialect:
		return "postgresql"
	case MySQLDialect:
		return "mysql"
	case SQLiteDialect:
		return "sqlite"
	case DuckDBDialect:
		return "duckdb"
	default:
		return ""
	}
}

// dialectAttrs prepends the dialect's db.system semantic-convention attribute
// so OTel-native APMs can group SQL spans by backend (mirroring the pebble
// and bbolt stores). Dialects without a canonical db.system value pass
// through unstamped.
func dialectAttrs(d Dialect, attrs []cqrsotel.KeyValue) []cqrsotel.KeyValue {
	if system := dbSystem(d); system != "" {
		attrs = append([]cqrsotel.KeyValue{cqrsotel.DBSystem(system)}, attrs...)
	}

	return attrs
}

// StartDialectSpan creates a client span for a SQL operation stamped with the
// dialect's db.system attribute (see dialectAttrs).
func StartDialectSpan(
	ctx context.Context,
	spanName string,
	d Dialect,
	attrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		Tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(dialectAttrs(d, attrs)...),
	)
}

// StartStreamSpanWithDialect is StartStreamSpan plus the dialect's db.system
// attribute.
func StartStreamSpanWithDialect(
	ctx context.Context,
	spanName string,
	d Dialect,
	ref id.StreamRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return StartDialectSpan(ctx, spanName, d,
		append(cqrsotel.StreamAttrs(ref.Type, ref.ID), extraAttrs...)...)
}

// StartSaveSpanWithDialect is StartSaveSpan plus the dialect's db.system
// attribute.
func StartSaveSpanWithDialect(
	ctx context.Context,
	spanName string,
	d Dialect,
	ref id.StreamRef,
	expectedVersion event.Version,
	eventCount int,
) (context.Context, cqrsotel.Span) {
	return StartDialectSpan(ctx, spanName, d,
		append(
			cqrsotel.StreamAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt(cqrsotel.AttrStreamVersion, expectedVersion.Int()),
			cqrsotel.AttrInt(cqrsotel.AttrEventCount, eventCount),
		)...)
}

// StartStreamSpan creates a span for a stream operation with stream attributes.
//
// Deprecated: use StartStreamSpanWithDialect so spans also carry db.system.
func StartStreamSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx,
		Tracer(),
		spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(
			append(cqrsotel.StreamAttrs(ref.Type, ref.ID), extraAttrs...)...,
		),
	)
}

// Deprecated: use StartStreamSpanWithDialect so spans also carry db.system.
func StartAggregateSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
	extraAttrs ...cqrsotel.KeyValue,
) (context.Context, cqrsotel.Span) {
	return StartStreamSpan(ctx, spanName, ref, extraAttrs...)
}

// StartSaveSpan creates a span for a save operation with stream attributes.
//
// Deprecated: use StartSaveSpanWithDialect so spans also carry db.system.
func StartSaveSpan(
	ctx context.Context,
	spanName string,
	ref id.StreamRef,
	expectedVersion event.Version,
	eventCount int,
) (context.Context, cqrsotel.Span) {
	return cqrsotel.StartSpan(
		ctx, Tracer(), spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(append(
			cqrsotel.StreamAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt(cqrsotel.AttrStreamVersion, expectedVersion.Int()),
			cqrsotel.AttrInt(cqrsotel.AttrEventCount, eventCount),
		)...),
	)
}
