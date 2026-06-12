package indexing

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

// Tracer returns the package-level OTel tracer for indexing operations.
// Uses the global TracerProvider; returns a no-op tracer when no
// provider is configured.
func Tracer() trace.Tracer {
	return cqrsotel.NewTracer("turso-indexing")
}

// Attribute keys for indexing telemetry.
const (
	AttrIndexName           = "indexing.index.name"
	AttrIndexTable          = "indexing.index.table"
	AttrIndexUnique         = "indexing.index.unique"
	AttrScanDetected        = "indexing.scan.detected"
	AttrRecommendationCount = "indexing.recommendation.count"
)

// SpanNames for common operations.
const (
	SpanAdvisorAnalyzeQuery   = "indexing.advisor.analyze_query"
	SpanAdvisorAnalyzeTable   = "indexing.advisor.analyze_table"
	SpanAdvisorMissingIndexes = "indexing.advisor.missing_indexes"
	SpanAutoIndexerApply      = "indexing.auto_indexer.apply"
	SpanAutoIndexerApplyCQRS  = "indexing.auto_indexer.apply_cqrs"
	SpanAutoIndexerDrop       = "indexing.auto_indexer.drop"
	SpanOptimizationsApply    = "indexing.optimizations.apply"
	SpanOptimizationsPragma   = "indexing.optimizations.pragma"
)

// startIndexingSpan starts a span with the indexing tracer and
// standard attributes. Returns the updated context and span.
func startIndexingSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return cqrsotel.StartSpan(
		ctx,
		Tracer(),
		spanName,
		trace.SpanKindClient,
		opts...,
	)
}

// endSpan ends a span and records any error.
func endSpan(span trace.Span, err error) {
	cqrsotel.EndWithError(span, err)
}

// recordIndexAttributes attaches standard index metadata to a span.
func recordIndexAttributes(span trace.Span, idx Index) {
	span.SetAttributes(
		attribute.String(AttrIndexName, idx.Name),
		attribute.String(AttrIndexTable, idx.Table),
		attribute.Bool(AttrIndexUnique, idx.Unique),
	)
}

// execAndTrace executes a DDL or DML statement, tracing it.
// The statement parameter is logged as an attribute for debugging.
// Use this for index creation, dropping, and ANALYZE statements.
func execAndTrace(
	ctx context.Context,
	db *sql.DB,
	spanName, sqlLabel string,
	ddl string,
) error {
	_, span := startIndexingSpan(
		ctx,
		spanName,
		trace.WithAttributes(attribute.String("db.statement", sqlLabel)),
	)
	defer endSpan(span, nil)

	_, err := db.ExecContext(ctx, ddl)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return fmt.Errorf("%s: %w", sqlLabel, err)
	}

	return nil
}
