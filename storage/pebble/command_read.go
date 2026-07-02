package pebble

import (
	"context"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
)

// Load retrieves all commands for an aggregate, ordered by ReceivedAt.
func (s *CommandStore) Load(
	ctx context.Context,
	ref command.AggregateRef,
) ([]*command.PersistedCommand, error) {
	_, span := startAggregateSpan(ctx, "pebble.command.load", ref)
	defer span.End()

	cmds, err := s.scanCommands(
		s.commandAggregatePrefix(ref),
		s.commandAggregateUpperBound(ref),
		0, "", nil,
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "pebble.command_load",
			"load commands for aggregate")
	}

	span.SetAttributes(cqrsotel.AttrInt("command.count", len(cmds)))

	return cmds, nil
}

// LoadFromTimestamp retrieves commands where ReceivedAt > after.
func (s *CommandStore) LoadFromTimestamp(
	ctx context.Context,
	ref command.AggregateRef,
	after time.Time,
) ([]*command.PersistedCommand, error) {
	_, span := startAggregateSpan(
		ctx,
		"pebble.command.load_from_timestamp",
		ref,
	)
	defer span.End()

	cmds, err := s.scanCommands(
		s.commandAggregatePrefix(ref),
		s.commandAggregateUpperBound(ref),
		0, "",
		func(cmd *command.PersistedCommand) bool { return cmd.ReceivedAt().After(after) },
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "pebble.command_load_from_timestamp",
			"load commands after timestamp")
	}

	return cmds, nil
}

// LoadToTimestamp retrieves commands where ReceivedAt <= maxTime.
func (s *CommandStore) LoadToTimestamp(
	ctx context.Context,
	ref command.AggregateRef,
	maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	_, span := startAggregateSpan(ctx, "pebble.command.load_to_timestamp", ref)
	defer span.End()

	cmds, err := s.scanCommands(
		s.commandAggregatePrefix(ref),
		s.commandAggregateUpperBound(ref),
		0, "",
		func(cmd *command.PersistedCommand) bool {
			return !cmd.ReceivedAt().After(maxTime)
		},
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "pebble.command_load_to_timestamp",
			"load commands up to timestamp")
	}

	return cmds, nil
}

// ReadAll returns all commands across all aggregates, ordered by command ID
// (which is ULID-based, so effectively time-ordered). Implements CommandJournal.
func (s *CommandStore) ReadAll(ctx context.Context) ([]*command.PersistedCommand, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.command.read_all",
		cqrsotel.SpanKindClient)
	defer span.End()

	cmds, err := s.scanCommands(
		[]byte(s.journalPrefix),
		[]byte(s.journalPrefix+"\xff"),
		0, "", nil,
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "pebble.command_read_all",
			"read all commands from journal")
	}

	span.SetAttributes(cqrsotel.AttrInt("command.count", len(cmds)))

	return cmds, nil
}

// ReadFrom returns commands after the given CommandID, ordered by ID.
// Implements SeekableCommandJournal for position-based command replay.
// Pass a zero CommandID to read from the beginning.
func (s *CommandStore) ReadFrom(
	ctx context.Context,
	afterCommandID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.command.read_from",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("limit", limit)))
	defer span.End()

	skipID := ""
	if !afterCommandID.IsZero() {
		skipID = afterCommandID.String()
	}

	cmds, err := s.scanCommands(
		[]byte(s.journalPrefix),
		[]byte(s.journalPrefix+"\xff"),
		limit, skipID, nil,
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "pebble.command_read_from",
			"read commands from journal after checkpoint")
	}

	span.SetAttributes(cqrsotel.AttrInt("command.count", len(cmds)))

	return cmds, nil
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
		return nil, event.WrapInfrastructure(err, "pebble.command_iter",
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
			return nil, event.WrapCorruption(err, "pebble.command_corrupt",
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
		return nil, event.WrapInfrastructure(err, "pebble.command_iter_error",
			"command iterator error")
	}

	return cmds, nil
}

// journalKeyCommandID extracts the command ID portion from a journal key.
func journalKeyCommandID(key []byte) string {
	for i := len(key) - 1; i >= 0; i-- { //nolint:modernize // reverse scan is clearer here
		if key[i] == ':' {
			return string(key[i+1:])
		}
	}

	return string(key)
}
