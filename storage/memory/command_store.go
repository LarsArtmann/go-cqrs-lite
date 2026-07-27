package memory

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// MemoryCommandStore is an in-memory implementation of command.Store.
// It is safe for concurrent use. Designed for testing and single-process deployments.
type MemoryCommandStore struct {
	dispatcher.Lifecycle

	mu             sync.RWMutex
	globalLog      []*command.PersistedCommand // canonical command storage
	streamIndex    map[string][]int            // streamKey → indices into globalLog
	commandIDIndex map[id.CommandID]int        // index into globalLog for duplicate detection
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
		streamIndex:    make(map[string][]int),
		commandIDIndex: make(map[id.CommandID]int),
	}
}

// Save persists a single command. Returns ErrDuplicateCommand if the command ID already exists.
func (s *MemoryCommandStore) Save(
	_ context.Context,
	ref command.StreamRef,
	cmd *command.PersistedCommand,
) error {
	return s.withWriteLock("memory.save_failed", "memory command store save", func() error {
		if dupErr := s.checkDuplicate(cmd.ID(), ""); dupErr != nil {
			return dupErr
		}

		s.appendCommand(ref.StreamKey(), cmd)

		return nil
	})
}

// withWriteLock checks the store is open, acquires the write lock, and runs fn
// under the lock. Centralises the wrapClosed + Lock + defer Unlock preamble
// shared by all write-side methods.
func (s *MemoryCommandStore) withWriteLock(code, msg string, fn func() error) error {
	if err := wrapClosed(s.CheckClosed(command.ErrStoreClosed), code, msg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return fn()
}

// AppendBatch appends multiple commands without duplicate checks on individual commands.
// If any command ID already exists, the entire batch fails.
func (s *MemoryCommandStore) AppendBatch(
	_ context.Context,
	ref command.StreamRef,
	cmds []*command.PersistedCommand,
) error {
	return s.withWriteLock("memory.append_batch_failed", "memory command store append batch", func() error {
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

			dupErr := s.checkDuplicate(cmd.ID(), " in batch")
			if dupErr != nil {
				return dupErr
			}
		}

		key := ref.StreamKey()
		for _, cmd := range cmds {
			s.appendCommand(key, cmd)
		}

		return nil
	})
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
	return s.loadFiltered(
		ref,
		"load from timestamp",
		func(cmds []*command.PersistedCommand) []*command.PersistedCommand {
			return filterByTimestampAfter(cmds, after)
		},
	)
}

// LoadToTimestamp retrieves commands where ReceivedAt <= maxTime.
func (s *MemoryCommandStore) LoadToTimestamp(
	_ context.Context,
	ref command.StreamRef,
	maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadFiltered(
		ref,
		"load to timestamp",
		func(cmds []*command.PersistedCommand) []*command.PersistedCommand {
			return filterByTimestampTo(cmds, maxTime)
		},
	)
}

// Close marks the store as closed. Subsequent operations return ErrStoreClosed.
func (s *MemoryCommandStore) Close() error {
	return s.Lifecycle.Close()
}

// ReadAll returns all commands across all streams, ordered by insertion
// (which matches ReceivedAt order since commands are appended on receipt).
// Implements command.CommandJournal.
func (s *MemoryCommandStore) ReadAll(_ context.Context) ([]*command.PersistedCommand, error) {
	if err := wrapClosed(
		s.CheckClosed(command.ErrStoreClosed),
		"memory.readall_failed",
		"memory command journal readall",
	); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.globalLog), nil
}

// ReadFrom returns commands after the given CommandID, ordered by insertion.
// Implements command.SeekableCommandJournal for position-based command replay.
func (s *MemoryCommandStore) ReadFrom(
	_ context.Context,
	afterCommandID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	if err := wrapClosed(
		s.CheckClosed(command.ErrStoreClosed),
		"memory.readfrom_failed",
		"memory command journal readfrom",
	); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	startIdx := 0

	if afterCommandID != (id.CommandID{}) {
		idx, exists := s.commandIDIndex[afterCommandID]
		if !exists {
			return nil, nil
		}

		startIdx = idx + 1
	}

	end := min(startIdx+limit, len(s.globalLog))

	if startIdx >= len(s.globalLog) {
		return nil, nil
	}

	return slices.Clone(s.globalLog[startIdx:end]), nil
}

func (s *MemoryCommandStore) checkDuplicate(cmdID id.CommandID, suffix string) error {
	if _, exists := s.commandIDIndex[cmdID]; exists {
		return errorfamily.WrapConflict(
			command.ErrDuplicateCommand,
			"memory.duplicate_command",
			fmt.Sprintf("command with ID %s already exists%s", cmdID, suffix),
		)
	}

	return nil
}

func (s *MemoryCommandStore) appendCommand(streamKey string, cmd *command.PersistedCommand) {
	idx := len(s.globalLog)
	s.commandIDIndex[cmd.ID()] = idx
	s.globalLog = append(s.globalLog, cmd)
	s.streamIndex[streamKey] = append(s.streamIndex[streamKey], idx)
}

func (s *MemoryCommandStore) loadFiltered(
	ref command.StreamRef,
	op string,
	filter func([]*command.PersistedCommand) []*command.PersistedCommand,
) ([]*command.PersistedCommand, error) {
	if err := wrapClosedf(
		s.CheckClosed(command.ErrStoreClosed),
		"memory.load_failed",
		"memory command store %s failed",
		op,
	); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ref.StreamKey()

	indices, exists := s.streamIndex[key]
	if !exists {
		return nil, errorfamily.WrapRejection(command.ErrCommandNotFound,
			"memory.command_not_found",
			fmt.Sprintf("memory %s stream %s not found", op, ref))
	}

	cmds := make([]*command.PersistedCommand, len(indices))
	for i, idx := range indices {
		cmds[i] = s.globalLog[idx]
	}

	if filter != nil {
		cmds = filter(cmds)
	}

	return slices.Clone(cmds), nil
}

func filterByTimestampAfter(
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

func filterByTimestampTo(
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
