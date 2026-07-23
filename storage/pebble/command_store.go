package pebble

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// CommandStore implements command.Store, command.CommandJournal, and
// command.SeekableCommandJournal backed by Pebble.
//
// Commands are append-only audit records ("who issued what command and when?").
// Each command is dual-written to two key spaces for efficient access from both
// access patterns:
//
//   - cqrs_command:{aggType}:{aggID}:{commandID} — per-aggregate index
//     Enables Load(ref), LoadFromTimestamp, LoadToTimestamp.
//
//   - cqrs_cmd_journal:{commandID} — global journal index
//     Enables ReadAll (full audit) and ReadFrom (incremental replay).
//
// CommandIDs are ULIDs (time-sortable), so journal keys are naturally ordered
// by receipt time — no separate timestamp index needed.
//
// The store shares the Pebble DB with other stores via disjoint key prefixes.
type CommandStore struct {
	storeBase

	journalPrefix string
}

// CommandStoreOption configures a CommandStore.
type CommandStoreOption func(*CommandStore)

// WithCommandAsyncWrites disables sync writes for higher throughput at the cost
// of durability guarantees. Use only when a lost command log on crash is
// acceptable (command logs are audit trails, not sources of truth).
func WithCommandAsyncWrites() CommandStoreOption {
	return func(s *CommandStore) { s.syncWrites = false }
}

// WithCommandPrefix overrides the default aggregate key prefix ("cqrs_command:").
// Useful when multiple logical command stores share one Pebble DB.
func WithCommandPrefix(p string) CommandStoreOption {
	return func(s *CommandStore) { s.prefix = p }
}

// NewCommandStore creates a new CommandStore using an existing Pebble DB.
// Returns ErrNilDatabase if db is nil.
func NewCommandStore(
	database *pebble.DB,
	logger *slog.Logger,
	opts ...CommandStoreOption,
) (*CommandStore, error) {
	if database == nil {
		return nil, ErrNilDatabase
	}

	s := &CommandStore{
		storeBase: storeBase{
			db:         database,
			logger:     logger,
			prefix:     "cqrs_command:",
			syncWrites: true,
		},
		journalPrefix: "cqrs_cmd_journal:",
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Close is a no-op; the underlying *pebble.DB is owned by the caller (or Backend).
// Implemented to satisfy io.Closer for command.CommandSink/CommandSource.
func (s *CommandStore) Close() error { return nil }

// Save persists a single command with duplicate-ID detection.
// Returns command.ErrDuplicateCommand if the command ID already exists.
func (s *CommandStore) Save(
	ctx context.Context,
	ref command.StreamRef,
	cmd *command.PersistedCommand,
) error {
	_, span := startAggregateSpan(ctx, "pebble.command.save", ref,
		cqrsotel.AttrString("command.type", string(cmd.Type())))
	defer span.End()

	jKey := s.commandJournalKey(cmd.ID())

	if s.commandExists(jKey) {
		return errorfamily.WrapConflict(command.ErrDuplicateCommand, "pebble.duplicate_command",
			fmt.Sprintf("command %s already exists", cmd.ID()))
	}

	data, err := s.serializeCommand(cmd)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapCorruption(err, "pebble.serialize_command",
			fmt.Sprintf("serialize command %s", cmd.ID()))
	}

	batch := s.db.NewBatch()
	defer func() { _ = batch.Close() }()

	err = s.writeCommandToBatch(batch, ref, cmd.ID(), jKey, data)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return err
	}

	err = batch.Commit(s.writeOptions())
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(err, "pebble.command_commit",
			fmt.Sprintf("commit command %s", cmd.ID()))
	}

	return nil
}

// AppendBatch appends multiple commands atomically.
// If any command ID already exists (in the batch or in the store), the entire
// batch fails — no partial writes.
func (s *CommandStore) AppendBatch(
	ctx context.Context,
	ref command.StreamRef,
	cmds []*command.PersistedCommand,
) error {
	if len(cmds) == 0 {
		return nil
	}

	_, span := startAggregateSpan(ctx, "pebble.command.append_batch", ref,
		cqrsotel.AttrInt("command.count", len(cmds)))
	defer span.End()

	batch := s.db.NewBatch()
	defer func() { _ = batch.Close() }()

	seen := make(map[id.CommandID]struct{}, len(cmds))

	for _, cmd := range cmds {
		if _, dup := seen[cmd.ID()]; dup {
			return errorfamily.WrapConflict(
				command.ErrDuplicateCommand,
				"pebble.batch_internal_dup",
				fmt.Sprintf("command %s appears multiple times in batch", cmd.ID()),
			)
		}

		seen[cmd.ID()] = struct{}{}

		jKey := s.commandJournalKey(cmd.ID())

		if s.commandExists(jKey) {
			return errorfamily.WrapConflict(
				command.ErrDuplicateCommand,
				"pebble.batch_existing_dup",
				fmt.Sprintf("command %s already exists", cmd.ID()),
			)
		}

		data, err := s.serializeCommand(cmd)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return errorfamily.WrapCorruption(err, "pebble.serialize_command_batch",
				fmt.Sprintf("serialize command %s", cmd.ID()))
		}

		err = s.writeCommandToBatch(batch, ref, cmd.ID(), jKey, data)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return err
		}
	}

	err := batch.Commit(s.writeOptions())
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(err, "pebble.command_batch_commit",
			fmt.Sprintf("commit batch of %d commands", len(cmds)))
	}

	return nil
}

func (s *CommandStore) commandExists(journalKey []byte) bool {
	return keyExists(s.db, journalKey)
}

func (s *CommandStore) writeCommandToBatch(
	batch *pebble.Batch,
	ref command.StreamRef,
	cmdID id.CommandID,
	journalKey, data []byte,
) error {
	aKey := s.commandKey(ref, cmdID)

	err := batch.Set(aKey, data, nil)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.command_aggregate_key",
			"add command to aggregate index")
	}

	err = batch.Set(journalKey, data, nil)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "pebble.command_journal_key",
			"add command to journal index")
	}

	return nil
}

// ── Key generation ──────────────────────────────────────────────────────────

func (s *CommandStore) commandKey(ref command.StreamRef, cmdID id.CommandID) []byte {
	return fmt.Appendf(nil, "%s%s:%s:%s", s.prefix, ref.Type, ref.ID, cmdID)
}

func (s *CommandStore) commandAggregatePrefix(ref command.StreamRef) []byte {
	return fmt.Appendf(nil, "%s%s:%s:", s.prefix, ref.Type, ref.ID)
}

func (s *CommandStore) commandAggregateUpperBound(ref command.StreamRef) []byte {
	return fmt.Appendf(nil, "%s%s:%s:\xff", s.prefix, ref.Type, ref.ID)
}

func (s *CommandStore) commandJournalKey(cmdID id.CommandID) []byte {
	return fmt.Appendf(nil, "%s%s", s.journalPrefix, cmdID)
}

// Ensure CommandStore implements command.Store, journal, and seekable journal.
var (
	_ command.Store                  = (*CommandStore)(nil)
	_ command.CommandJournal         = (*CommandStore)(nil)
	_ command.SeekableCommandJournal = (*CommandStore)(nil)
	_ io.Closer                      = (*CommandStore)(nil)
)
