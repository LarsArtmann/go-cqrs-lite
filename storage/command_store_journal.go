package storage

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// commandJournalReader builds a fresh JournalReader over the commands table.
// Construction is cheap (just captures method values); callers do not need to
// cache it.
func (s *SQLCommandStore) commandJournalReader() *sqlpkg.JournalReader[*command.PersistedCommand] {
	return &sqlpkg.JournalReader[*command.PersistedCommand]{
		DB:          s.DB,
		Dialect:     s.Dialect,
		CheckClosed: s.checkClosed,

		SpanNameAll:  "command.store.read_all",
		SpanNameFrom: "command.store.read_from",
		CountAttr:    "command.count",

		ErrCodeAll:        "storage.query_all_commands",
		ErrCodeReadFrom:   "storage.command_read_from",
		ErrCodeFromStart:  "storage.read_from_start",
		ErrCodeQueryStart: "storage.query_from_start",
		ErrCodeScan:       "storage.scan_from_position",

		EntityNoun:       "command",
		EntityNounPlural: "commands",

		Table:           sqlpkg.TableCommands,
		AllColumns:      sqlpkg.CommandColumns,
		PositionColumns: "e.id, e.command_type, e.aggregate_type, e.aggregate_id, e.payload, e.metadata, e.received_at",
		TimestampColumn: "received_at",

		Scan: s.scanCommands,
	}
}

// ReadAll returns all commands across all aggregates, ordered by received_at.
// Implements command.CommandJournal.
func (s *SQLCommandStore) ReadAll(ctx context.Context) ([]*command.PersistedCommand, error) {
	return s.commandJournalReader().ReadAll(ctx)
}

// ReadFrom returns commands after the given CommandID, ordered by received_at.
// Implements command.SeekableCommandJournal for position-based command replay.
func (s *SQLCommandStore) ReadFrom(
	ctx context.Context,
	afterCommandID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	afterID := ""
	if !afterCommandID.IsZero() {
		afterID = afterCommandID.String()
	}

	return s.commandJournalReader().ReadFrom(ctx, afterID, limit)
}
