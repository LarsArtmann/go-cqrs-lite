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
		cqrsotel.WithAttributes(cqrsotel.AggregateAttrs(ref.Type, ref.ID)...),
	)
	defer span.End()

	return sqlpkg.RunInTx(ctx, s.DB, span, func(tx *sql.Tx) error {
		err := s.insertCommand(ctx, tx, ref, cmd)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return errorfamily.WrapInfrastructure(err, "storage.insert_command",
				fmt.Sprintf("insert command %s for %s", cmd.Type(), ref))
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
			cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			cqrsotel.AttrInt("command.count", len(cmds)),
		)...),
	)
	defer span.End()

	return sqlpkg.RunInTx(ctx, s.DB, span, func(tx *sql.Tx) error {
		for _, cmd := range cmds {
			err := s.insertCommand(ctx, tx, ref, cmd)
			if err != nil {
				cqrsotel.RecordError(span, err)

				return errorfamily.WrapInfrastructure(err, "storage.insert_command",
					fmt.Sprintf("insert command %s for %s", cmd.Type(), ref))
			}
		}

		return nil
	})
}

func (s *SQLCommandStore) insertCommand(
	ctx context.Context,
	tx *sql.Tx,
	ref command.StreamRef,
	cmd *command.PersistedCommand,
) error {
	ph := make([]string, 7)
	for i := range 7 {
		ph[i] = s.Dialect.Placeholder(i + 1)
	}

	insertSQL := fmt.Sprintf(
		`INSERT INTO `+sqlpkg.TableCommands+` (id, command_type, aggregate_type, aggregate_id, payload, metadata, received_at)
		VALUES (%s, %s, %s, %s, %s, %s, %s)`,
		ph[0],
		ph[1],
		ph[2],
		ph[3],
		ph[4],
		ph[5],
		ph[6],
	)

	metadata, err := sqlpkg.MarshalMetadata(cmd.Metadata())
	if err != nil {
		return errorfamily.WrapCorruption(err, "storage.marshal_metadata",
			"marshal metadata for command "+string(cmd.Type()))
	}

	_, err = tx.ExecContext(
		ctx,
		insertSQL,
		cmd.ID(),
		string(cmd.Type()),
		string(ref.Type),
		ref.ID,
		cmd.Payload(),
		metadata,
		s.Dialect.FormatTime(cmd.ReceivedAt()),
	)
	if err != nil {
		if sqlpkg.IsDuplicateKeyError(err) {
			return errorfamily.WrapConflict(
				command.ErrDuplicateCommand,
				"storage.duplicate_command",
				fmt.Sprintf("command with ID %s already exists", cmd.ID()),
			)
		}

		return errorfamily.WrapInfrastructure(err, "storage.insert_command",
			"insert command "+string(cmd.Type()))
	}

	return nil
}
