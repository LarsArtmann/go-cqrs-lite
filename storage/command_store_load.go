package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

const commandCountAttr = "command.count"

var commandQueryConfig = sqlpkg.QueryConfig[*command.PersistedCommand]{ //nolint:gochecknoglobals // contains closures, cannot be const
	Columns:  sqlpkg.CommandColumns,
	Table:    sqlpkg.TableCommands,
	ScanRows: nil, // set per-store in loadWithSpan
	WrapError: func(err error, code, msg string) error {
		return event.WrapInfrastructure(err, code, msg)
	},
	WrapEmpty: func(err error, code, msg string) error {
		return event.WrapRejection(err, code, msg)
	},
	NotFound:   command.ErrCommandNotFound,
	DomainNoun: "commands",
}

// Load retrieves all commands for an aggregate, ordered by received_at.
func (s *SQLCommandStore) Load(
	ctx context.Context,
	ref command.AggregateRef,
) ([]*command.PersistedCommand, error) {
	return s.loadWithSpan(ctx, ref, sqlpkg.LoadParams{
		SpanName: "command.store.load", Attrs: cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		Where: "ORDER BY received_at ASC", RequireHit: true, ErrMsg: "query commands",
		CountAttr: commandCountAttr,
	})
}

// LoadFromTimestamp retrieves commands where ReceivedAt > after, ordered by received_at.
func (s *SQLCommandStore) LoadFromTimestamp(
	ctx context.Context,
	ref command.AggregateRef,
	after time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadWithSpan(ctx, ref, sqlpkg.LoadParams{
		SpanName: "command.store.load_from_timestamp",
		Attrs:    cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		Where: fmt.Sprintf(
			"AND received_at > %s ORDER BY received_at ASC",
			s.Dialect.Placeholder(3),
		),
		ExtraArgs: []any{s.Dialect.FormatTime(after)}, RequireHit: false,
		ErrMsg: "query commands from timestamp", CountAttr: commandCountAttr,
	})
}

// LoadToTimestamp retrieves commands where ReceivedAt <= maxTime, ordered by received_at.
func (s *SQLCommandStore) LoadToTimestamp(
	ctx context.Context,
	ref command.AggregateRef,
	maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadWithSpan(ctx, ref, sqlpkg.LoadParams{
		SpanName: "command.store.load_to_timestamp",
		Attrs:    cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		Where: fmt.Sprintf(
			"AND received_at <= %s ORDER BY received_at ASC",
			s.Dialect.Placeholder(3),
		),
		ExtraArgs: []any{s.Dialect.FormatTime(maxTime)}, RequireHit: true,
		ErrMsg: "query commands to timestamp", CountAttr: commandCountAttr,
	})
}

func (s *SQLCommandStore) loadWithSpan(
	ctx context.Context,
	ref command.AggregateRef,
	p sqlpkg.LoadParams,
) ([]*command.PersistedCommand, error) {
	cfg := commandQueryConfig
	cfg.ScanRows = s.scanCommands
	return sqlpkg.LoadWithSpan(ctx, s.DB, s.Dialect, s.checkClosed, cfg, p,
		string(ref.Type), ref.ID)
}
