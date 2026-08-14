package storage

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// Save persists a single command.
// Returns ErrDuplicateCommand if the command ID already exists (PRIMARY KEY violation).
func (s *SQLCommandStore) Save(
	ctx context.Context,
	ref command.StreamRef,
	cmd *command.PersistedCommand,
) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"command.store.save",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.StreamAttrs(ref.Type, ref.ID)...),
	)
	defer span.End()

	return sqlpkg.RunInTx(ctx, s.DB, span, func(tx *sql.Tx) error {
		if err := s.commandInserter(ref).Insert(ctx, tx, cmd); err != nil {
			cqrsotel.RecordError(span, err)

			return err
		}

		return nil
	})
}

// AppendBatch appends multiple commands in a single transaction.
// If any command ID already exists, the entire batch fails.
func (s *SQLCommandStore) AppendBatch(
	ctx context.Context,
	ref command.StreamRef,
	cmds []*command.PersistedCommand,
) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	if len(cmds) == 0 {
		return nil
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"command.store.append_batch",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(append(
			cqrsotel.StreamAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt("command.count", len(cmds)),
		)...),
	)
	defer span.End()

	return sqlpkg.RunInTx(ctx, s.DB, span, func(tx *sql.Tx) error {
		if err := s.commandInserter(ref).InsertAll(ctx, tx, cmds); err != nil {
			cqrsotel.RecordError(span, err)

			return err
		}

		return nil
	})
}

// commandInserter builds the shared write path for the commands table. It is
// constructed per call so RowArgs can capture the caller's stream ref, which
// is what the table's aggregate_type/aggregate_id columns record.
func (s *SQLCommandStore) commandInserter(ref command.StreamRef) *sqlpkg.Inserter[*command.PersistedCommand] {
	return &sqlpkg.Inserter[*command.PersistedCommand]{
		Dialect: s.Dialect,
		Table:   sqlpkg.TableCommands,
		Columns: []string{
			"id", "command_type", "aggregate_type", "aggregate_id",
			"payload", "metadata", "received_at",
		},
		EntityNoun:     "command",
		MarshalErrCode: "storage.marshal_metadata",
		InsertErrCode:  "storage.insert_command",
		Describe: func(cmd *command.PersistedCommand) string {
			return fmt.Sprintf("%s for %s", cmd.Type(), ref)
		},
		RowArgs: func(cmd *command.PersistedCommand) ([]any, error) {
			metadata, err := sqlpkg.MarshalMetadata(cmd.Metadata())
			if err != nil {
				return nil, err
			}

			return []any{
				cmd.ID(),
				string(cmd.Type()),
				string(ref.Type),
				ref.ID,
				cmd.Payload(),
				metadata,
				s.Dialect.FormatTime(cmd.ReceivedAt()),
			}, nil
		},
		Duplicate: func(err error, cmd *command.PersistedCommand) error {
			return errorfamily.WrapConflict(
				command.ErrDuplicateCommand,
				"storage.duplicate_command",
				fmt.Sprintf("command with ID %s already exists", cmd.ID()),
			)
		},
	}
}
