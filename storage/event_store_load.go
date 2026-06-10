package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

var eventQueryConfig = sqlpkg.QueryConfig[event.Event]{ //nolint:gochecknoglobals // contains closures, cannot be const
	Columns:  sqlpkg.EventColumns,
	Table:    sqlpkg.TableEvents,
	ScanRows: nil, // set per-store in loadWithSpan
	WrapError: func(err error, code, msg string) error {
		return event.WrapInfrastructure(err, code, msg)
	},
	WrapEmpty: func(err error, code, msg string) error {
		return event.WrapRejection(err, code, msg)
	},
	NotFound:   event.ErrAggregateNotFound,
	DomainNoun: "events",
}

func (s *SQLEventStore) loadWithSpan(
	ctx context.Context,
	ref event.AggregateRef,
	p sqlpkg.LoadParams,
) ([]event.Event, error) {
	cfg := eventQueryConfig
	cfg.ScanRows = s.scanEvents
	return sqlpkg.LoadWithSpan(ctx, s.DB, s.Dialect, s.checkClosed, cfg, p,
		string(ref.Type), ref.ID)
}

func (s *SQLEventStore) loadSimple(
	ctx context.Context,
	ref event.AggregateRef,
	spanName, order, errMsg string,
) ([]event.Event, error) {
	return s.loadWithSpan(ctx, ref, sqlpkg.LoadParams{
		SpanName: spanName, Attrs: cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		Where: order, RequireHit: true, ErrMsg: errMsg,
		CountAttr: cqrsotel.AttrEventCount,
	})
}

func (s *SQLEventStore) Load(ctx context.Context, ref event.AggregateRef) ([]event.Event, error) {
	return s.loadSimple(ctx, ref, "event.store.load", "ORDER BY version ASC", "query events")
}

func (s *SQLEventStore) LoadFromVersion(
	ctx context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	return s.loadWithSpan(ctx, ref, sqlpkg.LoadParams{
		SpanName: "event.store.load_from_version",
		Attrs: append(cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, version.Int())),
		Where:     fmt.Sprintf("AND version > %s ORDER BY version ASC", s.Dialect.Placeholder(3)),
		ExtraArgs: []any{version.Int()}, RequireHit: false, ErrMsg: "query events from version",
		CountAttr: cqrsotel.AttrEventCount,
	})
}

func (s *SQLEventStore) LoadToVersion(
	ctx context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	return s.loadWithSpan(ctx, ref, sqlpkg.LoadParams{
		SpanName: "event.store.load_to_version",
		Attrs: append(cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, maxVersion.Int())),
		Where:     fmt.Sprintf("AND version <= %s ORDER BY version ASC", s.Dialect.Placeholder(3)),
		ExtraArgs: []any{maxVersion.Int()}, RequireHit: true, ErrMsg: "query events to version",
		CountAttr: cqrsotel.AttrEventCount,
	})
}

func (s *SQLEventStore) LoadToTimestamp(
	ctx context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	return s.loadWithSpan(ctx, ref, sqlpkg.LoadParams{
		SpanName: "event.store.load_to_timestamp",
		Attrs:    cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		Where: fmt.Sprintf(
			"AND occurred_at <= %s ORDER BY version ASC",
			s.Dialect.Placeholder(3),
		),
		ExtraArgs:  []any{s.Dialect.FormatTime(maxTime)},
		RequireHit: true, ErrMsg: "query events to timestamp",
		CountAttr: cqrsotel.AttrEventCount,
	})
}

func (s *SQLEventStore) LoadBackwards(
	ctx context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	return s.loadSimple(
		ctx, ref,
		"event.store.load_backwards",
		"ORDER BY version DESC",
		"query events backwards",
	)
}
