package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// Save persists a single command.
// Returns ErrDuplicateCommand if the command ID already exists (PRIMARY KEY violation).
func (s *SQLCommandStore) Save(
	ctx context.Context,
	ref command.AggregateRef,
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

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return command.WrapInfrastructure(err, "storage.begin_tx", "begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	err = s.insertCommand(ctx, tx, ref, cmd)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return command.WrapInfrastructure(err, "storage.insert_command",
			fmt.Sprintf("insert command %s for %s", cmd.Type(), ref))
	}

	err = sqlpkg.CommitTx(tx)
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return err
}

// AppendBatch appends multiple commands in a single transaction.
// If any command ID already exists, the entire batch fails.
func (s *SQLCommandStore) AppendBatch(
	ctx context.Context,
	ref command.AggregateRef,
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

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return command.WrapInfrastructure(err, "storage.begin_tx", "begin transaction")
	}

	defer func() {
		_ = tx.Rollback()
	}()

	for _, cmd := range cmds {
		err = s.insertCommand(ctx, tx, ref, cmd)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return command.WrapInfrastructure(err, "storage.insert_command",
				fmt.Sprintf("insert command %s for %s", cmd.Type(), ref))
		}
	}

	err = sqlpkg.CommitTx(tx)
	if err != nil {
		cqrsotel.RecordError(span, err)
	}

	return err
}

func (s *SQLCommandStore) insertCommand(
	ctx context.Context,
	tx *sql.Tx,
	ref command.AggregateRef,
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
		return command.WrapCorruption(err, "storage.marshal_metadata",
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
		if isDuplicateKeyError(err) {
			return command.WrapConflict(
				command.ErrDuplicateCommand,
				"storage.duplicate_command",
				fmt.Sprintf("command with ID %s already exists", cmd.ID()),
			)
		}

		return command.WrapInfrastructure(err, "storage.insert_command",
			"insert command "+string(cmd.Type()))
	}

	return nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}
