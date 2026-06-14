package sql

import (
	"context"
	"database/sql"
	"fmt"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
)

// LoadParams holds the parameters for a parameterized load query.
type LoadParams struct {
	SpanName   string
	Attrs      []cqrsotel.KeyValue
	Where      string
	ExtraArgs  []any
	RequireHit bool
	ErrMsg     string
	CountAttr  string
}

// QueryConfig parameterizes a domain-specific load query with type-specific
// scanning and error wrapping, enabling shared query infrastructure for
// event and command stores.
type QueryConfig[T any] struct {
	Columns    string
	Table      string
	ScanRows   func(*sql.Rows) ([]T, error)
	WrapError  func(error, string, string) error
	WrapEmpty  func(error, string, string) error
	NotFound   error
	DomainNoun string
}

// LoadWithSpan executes a traced, parameterized load query with closed-store
// checking and OpenTelemetry span creation.
func LoadWithSpan[T any](
	ctx context.Context,
	db *sql.DB,
	d Dialect,
	checkClosed func() error,
	cfg QueryConfig[T],
	p LoadParams,
	aggType string,
	aggID any,
) ([]T, error) {
	if err := checkClosed(); err != nil {
		return nil, fmt.Errorf("load %s %v: %w", aggType, aggID, err)
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, Tracer(), p.SpanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(p.Attrs...),
	)
	defer span.End()

	results, err := QueryRows(ctx, db, d, cfg, p, aggType, aggID)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, fmt.Errorf("query %s %v: %w", aggType, aggID, err)
	}

	span.SetAttributes(cqrsotel.AttrInt(p.CountAttr, len(results)))

	return results, nil
}

// QueryRows executes a parameterized query, scans results, and enforces
// non-empty requirements.
func QueryRows[T any](
	ctx context.Context,
	db *sql.DB,
	d Dialect,
	cfg QueryConfig[T],
	p LoadParams,
	aggType string,
	aggID any,
) ([]T, error) {
	p1, p2 := d.Placeholder(1), d.Placeholder(2)
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE aggregate_type = %s AND aggregate_id = %s %s",
		cfg.Columns, cfg.Table, p1, p2, p.Where,
	)

	args := make([]any, 0, 2+len(p.ExtraArgs))
	args = append(args, aggType, aggID)
	args = append(args, p.ExtraArgs...)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, cfg.WrapError(err, "storage.query_"+cfg.Table,
			p.ErrMsg+fmt.Sprintf(" (%s=%v, where=%s)", aggType, aggID, p.Where))
	}
	defer func() { _ = rows.Close() }()

	results, err := cfg.ScanRows(rows)
	if err != nil {
		return nil, cfg.WrapError(err, "storage.scan_"+cfg.Table,
			p.ErrMsg+fmt.Sprintf(" (%s=%v, where=%s)", aggType, aggID, p.Where))
	}

	if err := rows.Err(); err != nil {
		return nil, cfg.WrapError(err, "storage.rows_err_"+cfg.Table,
			p.ErrMsg+fmt.Sprintf(" (%s=%v, where=%s)", aggType, aggID, p.Where))
	}

	if p.RequireHit && len(results) == 0 {
		return nil, cfg.WrapEmpty(cfg.NotFound, "storage.not_found",
			fmt.Sprintf("no %s found for %s %v", cfg.DomainNoun, aggType, aggID))
	}

	return results, nil
}
