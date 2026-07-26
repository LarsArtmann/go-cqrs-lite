package pebble

import (
	"context"
	"time"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// Load retrieves all commands for a stream, ordered by ReceivedAt.
func (s *CommandStore) Load(
	ctx context.Context,
	ref command.StreamRef,
) ([]*command.PersistedCommand, error) {
	_, span := startStreamSpan(ctx, "pebble.command.load", ref)
	defer span.End()

	cmds, err := s.scanCommands(
		s.commandStreamPrefix(ref),
		s.commandStreamUpperBound(ref),
		0, "", nil,
	)

	return finalizeScan(span, cmds, err, "pebble.command_load",
		"load commands for stream", "command.count")
}

// LoadFromTimestamp retrieves commands where ReceivedAt > after.
func (s *CommandStore) LoadFromTimestamp(
	ctx context.Context,
	ref command.StreamRef,
	after time.Time,
) ([]*command.PersistedCommand, error) {
	_, span := startStreamSpan(
		ctx,
		"pebble.command.load_from_timestamp",
		ref,
	)
	defer span.End()

	cmds, err := s.scanCommands(
		s.commandStreamPrefix(ref),
		s.commandStreamUpperBound(ref),
		0, "",
		func(cmd *command.PersistedCommand) bool { return cmd.ReceivedAt().After(after) },
	)
	if err != nil {
		return nil, reportScanErr(span, err, "pebble.command_load_from_timestamp",
			"load commands after timestamp")
	}

	return cmds, nil
}

// LoadToTimestamp retrieves commands where ReceivedAt <= maxTime.
func (s *CommandStore) LoadToTimestamp(
	ctx context.Context,
	ref command.StreamRef,
	maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	_, span := startStreamSpan(ctx, "pebble.command.load_to_timestamp", ref)
	defer span.End()

	cmds, err := s.scanCommands(
		s.commandStreamPrefix(ref),
		s.commandStreamUpperBound(ref),
		0, "",
		func(cmd *command.PersistedCommand) bool {
			return !cmd.ReceivedAt().After(maxTime)
		},
	)
	if err != nil {
		return nil, reportScanErr(span, err, "pebble.command_load_to_timestamp",
			"load commands up to timestamp")
	}

	return cmds, nil
}

// ReadAll returns all commands across all streams, ordered by command ID
// (which is ULID-based, so effectively time-ordered). Implements CommandJournal.
func (s *CommandStore) ReadAll(ctx context.Context) ([]*command.PersistedCommand, error) {
	span := startReadSpan(ctx, "pebble.command.read_all")
	defer span.End()

	cmds, err := s.scanCommands(
		[]byte(s.journalPrefix),
		[]byte(s.journalPrefix+"\xff"),
		0, "", nil,
	)

	return finalizeScan(span, cmds, err, "pebble.command_read_all",
		"read all commands from journal", "command.count")
}

// ReadFrom returns commands after the given CommandID, ordered by ID.
// Implements SeekableCommandJournal for position-based command replay.
// Pass a zero CommandID to read from the beginning.
func (s *CommandStore) ReadFrom(
	ctx context.Context,
	afterCommandID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	span := startLimitSpan(ctx, "pebble.command.read_from", limit)
	defer span.End()

	skipID := idOrEmpty(afterCommandID)

	cmds, err := s.scanCommands(
		[]byte(s.journalPrefix),
		[]byte(s.journalPrefix+"\xff"),
		limit, skipID, nil,
	)

	return finalizeScan(span, cmds, err, "pebble.command_read_from",
		"read commands from journal after checkpoint", "command.count")
}

// scanCommands iterates over the given key range, deserializing commands.
//
// Parameters:
//   - limit: 0 means no limit.
//   - skipUntilID: if non-empty, skip all entries until the one whose journal
//     key ends with this ID is found (inclusive — that entry is also skipped).
//     Used by ReadFrom to resume after a checkpoint.
//   - filter: optional predicate; entries where filter returns false are skipped.
func (s *CommandStore) scanCommands(
	lowerBound, upperBound []byte,
	limit int,
	skipUntilID string,
	filter func(*command.PersistedCommand) bool,
) ([]*command.PersistedCommand, error) {
	iter, err := s.db.NewIter(
		&pebble.IterOptions{
			LowerBound: lowerBound,
			UpperBound: upperBound,
		},
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "pebble.command_iter",
			"create command iterator")
	}

	defer func() { _ = iter.Close() }()

	skipping := skipUntilID != ""

	var cmds []*command.PersistedCommand

	for iter.First(); iter.Valid(); iter.Next() {
		if skipping {
			if journalKeyCommandID(iter.Key()) == skipUntilID {
				skipping = false
			}

			continue
		}

		cmd, err := s.deserializeCommand(iter.Value())
		if err != nil {
			return nil, errorfamily.WrapCorruption(err, "pebble.command_corrupt",
				"corrupt command at key "+string(iter.Key()))
		}

		if filter != nil && !filter(cmd) {
			continue
		}

		cmds = append(cmds, cmd)

		if limit > 0 && len(cmds) >= limit {
			break
		}
	}

	err = checkIteratorError(iter)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "pebble.command_iter_error",
			"command iterator error")
	}

	return cmds, nil
}

// journalKeyCommandID extracts the command ID portion from a journal key.
func journalKeyCommandID(key []byte) string {
	return lastSegmentAfterByte(key, ':')
}
