package storage

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// ReadAll returns all commands across all aggregates, ordered by received_at.
// Implements command.CommandJournal.
func (s *SQLCommandStore) ReadAll(ctx context.Context) ([]*command.PersistedCommand, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"command.store.read_all",
		cqrsotel.SpanKindClient,
	)
	defer span.End()

	query := `SELECT ` + sqlpkg.CommandColumns + `
		FROM ` + sqlpkg.TableCommands + ` ORDER BY received_at ASC`

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(
			err,
			"storage.query_all_commands",
			"query all commands",
		)
	}
	defer func() { _ = rows.Close() }()

	cmds, scanErr := s.scanCommands(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
	}

	span.SetAttributes(cqrsotel.AttrInt("command.count", len(cmds)))

	return cmds, scanErr
}

// ReadFrom returns commands after the given CommandID, ordered by received_at.
// Implements command.SeekableCommandJournal for position-based command replay.
func (s *SQLCommandStore) ReadFrom(
	ctx context.Context,
	afterCommandID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	if err := s.checkClosed(); err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, "storage.command_read_from",
			"read from command store (limit=%d, after=%s)", limit, afterCommandID)
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"command.store.read_from",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("cqrs.journal.limit", limit)),
	)
	defer span.End()

	if afterCommandID.IsZero() {
		cmds, err := s.loadCommandsFromStart(ctx, limit)
		if err != nil {
			cqrsotel.RecordError(span, err)
			return cmds, errorfamily.WrapInfrastructure(err, "storage.read_from_start",
				fmt.Sprintf("read commands from start (limit=%d)", limit))
		}
		span.SetAttributes(cqrsotel.AttrInt("command.count", len(cmds)))
		return cmds, nil
	}

	p1 := s.Dialect.Placeholder(1)
	p2 := s.Dialect.Placeholder(2)
	p3 := s.Dialect.Placeholder(3)
	query := fmt.Sprintf(
		`SELECT e.id, e.command_type, e.aggregate_type, e.aggregate_id, e.payload, e.metadata, e.received_at
		FROM `+sqlpkg.TableCommands+` e
		JOIN `+sqlpkg.TableCommands+` c ON c.id = %s
		WHERE (e.received_at > c.received_at) OR (e.received_at = c.received_at AND e.id > %s)
		ORDER BY e.received_at ASC, e.id ASC`,
		p1,
		p2,
	)
	args := []any{afterCommandID.String(), afterCommandID.String()}
	if limit > 0 {
		query += " LIMIT " + p3
		args = append(args, limit)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(err, "storage.query_from_position",
			fmt.Sprintf("query commands from position (limit=%d)", limit))
	}
	defer func() { _ = rows.Close() }()

	cmds, scanErr := s.scanCommands(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
		return cmds, errorfamily.WrapInfrastructure(scanErr, "storage.scan_from_position",
			fmt.Sprintf("scan commands from position (limit=%d)", limit))
	}
	span.SetAttributes(cqrsotel.AttrInt("command.count", len(cmds)))

	return cmds, nil
}

func (s *SQLCommandStore) loadCommandsFromStart(
	ctx context.Context,
	limit int,
) ([]*command.PersistedCommand, error) {
	if limit <= 0 {
		return s.ReadAll(ctx)
	}

	p1 := s.Dialect.Placeholder(1)
	query := `SELECT ` + sqlpkg.CommandColumns + `
		FROM ` + sqlpkg.TableCommands + ` ORDER BY received_at ASC LIMIT ` + p1

	rows, err := s.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.query_from_start",
			fmt.Sprintf("query commands from start (limit=%d)", limit))
	}
	defer func() { _ = rows.Close() }()

	return s.scanCommands(rows)
}
