package storage

import (
	"context"
	"fmt"
	"time"


	"github.com/larsartmann/go-cqrs-lite/command/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// Load retrieves all commands for an aggregate, ordered by received_at.
func (s *SQLCommandStore) Load(
	ctx context.Context,
	ref command.AggregateRef,
) ([]*command.PersistedCommand, error) {
	return s.loadWithSpan(ctx, ref, loadCommandParams{
		spanName:   "command.store.load",
		attrs:      cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		where:      "ORDER BY received_at ASC",
		requireHit: true,
		errMsg:     "query commands",
	})
}

// LoadFromTimestamp retrieves commands where ReceivedAt > after, ordered by received_at.
func (s *SQLCommandStore) LoadFromTimestamp(
	ctx context.Context,
	ref command.AggregateRef,
	after time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadWithSpan(ctx, ref, loadCommandParams{
		spanName: "command.store.load_from_timestamp",
		attrs:    cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		where: fmt.Sprintf(
			"AND received_at > %s ORDER BY received_at ASC",
			s.Dialect.Placeholder(3),
		),
		extraArgs:  []any{s.Dialect.FormatTime(after)},
		requireHit: false,
		errMsg:     "query commands from timestamp",
	})
}

// LoadToTimestamp retrieves commands where ReceivedAt <= maxTime, ordered by received_at.
func (s *SQLCommandStore) LoadToTimestamp(
	ctx context.Context,
	ref command.AggregateRef,
	maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadWithSpan(ctx, ref, loadCommandParams{
		spanName: "command.store.load_to_timestamp",
		attrs:    cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		where: fmt.Sprintf(
			"AND received_at <= %s ORDER BY received_at ASC",
			s.Dialect.Placeholder(3),
		),
		extraArgs:  []any{s.Dialect.FormatTime(maxTime)},
		requireHit: true,
		errMsg:     "query commands to timestamp",
	})
}

type loadCommandParams struct {
	spanName   string
	attrs      []cqrsotel.KeyValue
	where      string
	extraArgs  []any
	requireHit bool
	errMsg     string
}

func (s *SQLCommandStore) loadWithSpan(
	ctx context.Context,
	ref command.AggregateRef,
	p loadCommandParams,
) ([]*command.PersistedCommand, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		p.spanName,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(p.attrs...),
	)
	defer span.End()

	cmds, err := s.queryCommands(ctx, ref, p.where, p.extraArgs, p.requireHit, p.errMsg)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, err
	}

	span.SetAttributes(cqrsotel.AttrInt("command.count", len(cmds)))

	return cmds, nil
}

func (s *SQLCommandStore) queryCommands(
	ctx context.Context,
	ref command.AggregateRef,
	whereSuffix string,
	extraArgs []any,
	requireNonEmpty bool,
	errMsg string,
) ([]*command.PersistedCommand, error) {
	p1, p2 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2)
	query := fmt.Sprintf(
		`SELECT `+sqlpkg.CommandColumns+`
		FROM `+sqlpkg.TableCommands+` WHERE aggregate_type = %s AND aggregate_id = %s %s`,
		p1,
		p2,
		whereSuffix,
	)
	args := make([]any, 0, 2+len(extraArgs))
	args = append(args, string(ref.Type), ref.ID)
	args = append(args, extraArgs...)

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, command.WrapInfrastructure(
			err,
			"storage.query_commands",
			errMsg+" (where="+whereSuffix+")",
		)
	}
	defer func() { _ = rows.Close() }()

	cmds, err := s.scanCommands(rows)
	if err != nil {
		return nil, command.WrapInfrastructure(
			err,
			"storage.scan_commands",
			errMsg+" (where="+whereSuffix+")",
		)
	}

	if requireNonEmpty && len(cmds) == 0 {
		return nil, command.WrapRejection(
			command.ErrCommandNotFound,
			"storage.command_not_found",
			fmt.Sprintf("no commands found for %s %s", ref.Type, ref.ID),
		)
	}

	return cmds, nil
}
