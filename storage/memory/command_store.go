package memory

import (
	"context"
	"fmt"
	"io"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// MemoryCommandStore is an in-memory implementation of command.Store.
// It is safe for concurrent use. Designed for testing and single-process deployments.
//
// It embeds the generic [LogStore] core; this file supplies only the
// command-specific policies (duplicate detection, stream-not-found errors).
type MemoryCommandStore struct {
*LogStore[*command.PersistedCommand, id.CommandID]
}

var (
	_ command.Store                  = (*MemoryCommandStore)(nil)
	_ command.CommandJournal         = (*MemoryCommandStore)(nil)
	_ command.SeekableCommandJournal = (*MemoryCommandStore)(nil)
	_ io.Closer                      = (*MemoryCommandStore)(nil)
)

// NewMemoryCommandStore creates a new in-memory command store.
func NewMemoryCommandStore() *MemoryCommandStore {
	return &MemoryCommandStore{
		LogStore: NewLogStore(LogStoreConfig[*command.PersistedCommand, id.CommandID]{
			GetID:     func(cmd *command.PersistedCommand) id.CommandID { return cmd.ID() },
			IsZeroID:  func(cmdID id.CommandID) bool { return cmdID == (id.CommandID{}) },
			ClosedErr: command.ErrStoreClosed,
			NewDupErr: func(cmdID id.CommandID, suffix string) error {
				return errorfamily.WrapConflict(
					command.ErrDuplicateCommand,
					"memory.duplicate_command",
					fmt.Sprintf("command with ID %s already exists%s", cmdID, suffix))
			},
			NewNotFound: func(op, streamKey string) error {
				return errorfamily.WrapRejection(command.ErrCommandNotFound,
					"memory.command_not_found",
					fmt.Sprintf("memory %s stream %s not found", op, streamKey))
			},
			TrackStreams: true,
		}),
	}
}

// Save persists a single command. Returns ErrDuplicateCommand if the command ID already exists.
func (s *MemoryCommandStore) Save(
	_ context.Context,
	ref command.StreamRef,
	cmd *command.PersistedCommand,
) error {
	return s.WithWrite("memory.save_failed", "memory command store save", func() error {
		if dupErr := s.CheckDuplicateLocked(cmd.ID(), ""); dupErr != nil {
			return dupErr
		}

		s.AppendLocked(ref.StreamKey(), []*command.PersistedCommand{cmd})

		return nil
	})
}

// AppendBatch appends multiple commands without duplicate checks on individual commands.
// If any command ID already exists, the entire batch fails.
func (s *MemoryCommandStore) AppendBatch(
	_ context.Context,
	ref command.StreamRef,
	cmds []*command.PersistedCommand,
) error {
	return s.WithWrite(
		"memory.append_batch_failed",
		"memory command store append batch",
		func() error {
			seen := make(map[id.CommandID]struct{}, len(cmds))

			for _, cmd := range cmds {
				if _, dup := seen[cmd.ID()]; dup {
					return errorfamily.WrapConflict(
						command.ErrDuplicateCommand,
						"memory.duplicate_command",
						fmt.Sprintf("command with ID %s appears multiple times in batch", cmd.ID()),
					)
				}

				seen[cmd.ID()] = struct{}{}

				dupErr := s.CheckDuplicateLocked(cmd.ID(), " in batch")
				if dupErr != nil {
					return dupErr
				}
			}

			s.AppendLocked(ref.StreamKey(), cmds)

			return nil
		},
	)
}

// Load retrieves all commands for a stream.
func (s *MemoryCommandStore) Load(
	_ context.Context,
	ref command.StreamRef,
) ([]*command.PersistedCommand, error) {
	return s.loadFiltered(ref, "load", nil)
}

// LoadFromTimestamp retrieves commands where ReceivedAt > after.
func (s *MemoryCommandStore) LoadFromTimestamp(
	_ context.Context,
	ref command.StreamRef,
	after time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadFiltered(ref, "load from timestamp",
		func(cmds []*command.PersistedCommand) []*command.PersistedCommand {
			return filterCommandsByTimestampAfter(cmds, after)
		})
}

// LoadToTimestamp retrieves commands where ReceivedAt <= maxTime.
func (s *MemoryCommandStore) LoadToTimestamp(
	_ context.Context,
	ref command.StreamRef,
	maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadFiltered(ref, "load to timestamp",
		func(cmds []*command.PersistedCommand) []*command.PersistedCommand {
			return filterCommandsByTimestampTo(cmds, maxTime)
		})
}

// ReadAll returns all commands across all streams, ordered by insertion
// (which matches ReceivedAt order since commands are appended on receipt).
// Implements command.CommandJournal.
func (s *MemoryCommandStore) ReadAll(_ context.Context) ([]*command.PersistedCommand, error) {
	return WithReadLock(s.LogStore,
		"memory.readall_failed",
		"memory command journal readall",
		func() ([]*command.PersistedCommand, error) {
			return s.ReadAllLocked(), nil
		},
	)
}

// ReadFrom returns commands after the given CommandID, ordered by insertion.
// A missing start position returns nothing — an unknown position means
// nothing new. Implements command.SeekableCommandJournal.
func (s *MemoryCommandStore) ReadFrom(
	_ context.Context,
	afterCommandID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	return WithReadLock(s.LogStore,
		"memory.readfrom_failed",
		"memory command journal readfrom",
		func() ([]*command.PersistedCommand, error) {
			return s.ReadFromLocked(afterCommandID, limit, false), nil
		},
	)
}

func (s *MemoryCommandStore) loadFiltered(
	ref command.StreamRef,
	op string,
	filter func([]*command.PersistedCommand) []*command.PersistedCommand,
) ([]*command.PersistedCommand, error) {
	return WithReadLock(
		s.LogStore,
		"memory.load_failed",
		fmt.Sprintf("memory command store %s failed", op),
		func() ([]*command.PersistedCommand, error) {
			return s.LoadStreamLocked(op, ref.StreamKey(), filter)
		},
	)
}

func filterCommandsByTimestampAfter(
	cmds []*command.PersistedCommand,
	after time.Time,
) []*command.PersistedCommand {
	var result []*command.PersistedCommand

	for _, cmd := range cmds {
		if cmd.ReceivedAt().After(after) {
			result = append(result, cmd)
		}
	}

	return result
}

func filterCommandsByTimestampTo(
	cmds []*command.PersistedCommand,
	maxTime time.Time,
) []*command.PersistedCommand {
	var result []*command.PersistedCommand

	for _, cmd := range cmds {
		if !cmd.ReceivedAt().After(maxTime) {
			result = append(result, cmd)
		}
	}

	return result
}
