package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// SQLCommandStore persists commands in a SQL database.
type SQLCommandStore struct {
	sqlpkg.Base

	ownDB  bool
	closed atomic.Bool
}

// NewSQLCommandStore creates a new PostgreSQL-backed command store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLCommandStore(db *sql.DB) (*SQLCommandStore, error) {
	return newSQLCommandStoreWithDialect(db, sqlpkg.PostgresDialect{})
}

// NewSQLiteCommandStore creates a new SQLite-backed command store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteCommandStore(db *sql.DB) (*SQLCommandStore, error) {
	return newSQLCommandStoreWithDialect(db, sqlpkg.SQLiteDialect{})
}

// NewSQLCommandStoreWithDialect creates a new SQL-backed command store with a custom dialect.
// This enables consumers to use any SQL backend (MySQL, CockroachDB, etc.) by implementing the Dialect interface.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLCommandStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLCommandStore, error) {
	return newSQLCommandStoreWithDialect(db, d)
}

func newSQLCommandStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLCommandStore, error) {
	base, err := sqlpkg.NewBase(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLCommandStore{Base: base}, nil
}

// Close closes the store. If WithOwnership was set, also closes the underlying *sql.DB.
func (s *SQLCommandStore) Close() error {
	s.closed.Store(true)

	if s.ownDB {
		return s.DB.Close()
	}

	return nil
}

func (s *SQLCommandStore) checkClosed() error {
	if s.closed.Load() {
		return command.ErrStoreClosed
	}

	return nil
}

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
		trace.SpanKindClient,
		trace.WithAttributes(cqrsotel.AggregateAttrs(ref.Type, ref.ID)...),
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
		trace.SpanKindClient,
		trace.WithAttributes(append(
			cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			attribute.Int("command.count", len(cmds)),
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
	attrs      []attribute.KeyValue
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
		trace.SpanKindClient,
		trace.WithAttributes(p.attrs...),
	)
	defer span.End()

	cmds, err := s.queryCommands(ctx, ref, p.where, p.extraArgs, p.requireHit, p.errMsg)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("command.count", len(cmds)))

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

var _ command.Store = (*SQLCommandStore)(nil)
