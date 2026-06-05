package memory

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/dispatcher/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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
	_ command.Store = (*MemoryCommandStore)(nil)
	_ io.Closer     = (*MemoryCommandStore)(nil)
)

// NewMemoryCommandStore creates a new in-memory command store.
func NewMemoryCommandStore() *MemoryCommandStore {
	//nolint:exhaustruct // embedded Lifecycle has unexported fields from different package
	return &MemoryCommandStore{
		streamIndex:    make(map[string][]int),
		commandIDIndex: make(map[id.CommandID]int),
	}
}

// Save persists a single command. Returns ErrDuplicateCommand if the command ID already exists.
func (s *MemoryCommandStore) Save(
	_ context.Context,
	ref command.AggregateRef,
	cmd *command.PersistedCommand,
) error {
	err := s.CheckClosed(command.ErrStoreClosed)
	if err != nil {
		return event.WrapInfrastructure(err, "memory.save_failed", "memory command store save")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.commandIDIndex[cmd.ID()]; exists {
		return event.WrapConflict(
			command.ErrDuplicateCommand,
			"memory.duplicate_command",
			fmt.Sprintf("command with ID %s already exists", cmd.ID()),
		)
	}

	s.appendCommand(ref.StreamKey(), cmd)

	return nil
}

// AppendBatch appends multiple commands without duplicate checks on individual commands.
// If any command ID already exists, the entire batch fails.
func (s *MemoryCommandStore) AppendBatch(
	_ context.Context,
	ref command.AggregateRef,
	cmds []*command.PersistedCommand,
) error {
	err := s.CheckClosed(command.ErrStoreClosed)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"memory.append_batch_failed",
			"memory command store append batch",
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[id.CommandID]struct{}, len(cmds))
	for _, cmd := range cmds {
		if _, dup := seen[cmd.ID()]; dup {
			return event.WrapConflict(
				command.ErrDuplicateCommand,
				"memory.duplicate_command",
				fmt.Sprintf("command with ID %s appears multiple times in batch", cmd.ID()),
			)
		}

		seen[cmd.ID()] = struct{}{}

		if _, exists := s.commandIDIndex[cmd.ID()]; exists {
			return event.WrapConflict(
				command.ErrDuplicateCommand,
				"memory.duplicate_command",
				fmt.Sprintf("command with ID %s already exists in batch", cmd.ID()),
			)
		}
	}

	key := ref.StreamKey()
	for _, cmd := range cmds {
		s.appendCommand(key, cmd)
	}

	return nil
}

// Load retrieves all commands for an aggregate.
func (s *MemoryCommandStore) Load(
	_ context.Context,
	ref command.AggregateRef,
) ([]*command.PersistedCommand, error) {
	return s.loadFiltered(ref, "load", nil)
}

// LoadFromTimestamp retrieves commands where ReceivedAt > after.
func (s *MemoryCommandStore) LoadFromTimestamp(
	_ context.Context,
	ref command.AggregateRef,
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
	ref command.AggregateRef,
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
	return s.Lifecycle.Close() //nolint:wrapcheck
}

func (s *MemoryCommandStore) appendCommand(streamKey string, cmd *command.PersistedCommand) {
	idx := len(s.globalLog)
	s.commandIDIndex[cmd.ID()] = idx
	s.globalLog = append(s.globalLog, cmd)
	s.streamIndex[streamKey] = append(s.streamIndex[streamKey], idx)
}

func (s *MemoryCommandStore) loadFiltered(
	ref command.AggregateRef,
	op string,
	filter func([]*command.PersistedCommand) []*command.PersistedCommand,
) ([]*command.PersistedCommand, error) {
	err := s.CheckClosed(command.ErrStoreClosed)
	if err != nil {
		return nil, event.Wrapf(
			err,
			event.Infrastructure,
			"memory.load_failed",
			"memory command store %s failed",
			op,
		)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ref.StreamKey()

	indices, exists := s.streamIndex[key]
	if !exists {
		return nil, fmt.Errorf(
			"memory %s aggregate %s: %w",
			op,
			ref,
			command.ErrCommandNotFound,
		)
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
