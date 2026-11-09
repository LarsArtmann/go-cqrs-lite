package eventstore

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// eventJournalReader builds a fresh JournalReader over the events table.
func (s *SQLEventStore) eventJournalReader() *sqlpkg.JournalReader[event.Event] {
	return &sqlpkg.JournalReader[event.Event]{
		DB:          s.DB,
		Dialect:     s.Dialect,
		CheckClosed: s.checkClosed,

		SpanNameAll:  "event.store.read_all",
		SpanNameFrom: "event.store.read_from",
		CountAttr:    cqrsotel.AttrEventCount,

		ErrCodeAll:        "storage.query_all_events",
		ErrCodeReadFrom:   "storage.event_read_from",
		ErrCodeFromStart:  "storage.read_from_start",
		ErrCodeQueryStart: "storage.query_from_start",
		ErrCodeScan:       "storage.scan_from_position",

		EntityNoun:       "event",
		EntityNounPlural: "events",

		Table:           sqlpkg.TableEvents,
		AllColumns:      sqlpkg.EventColumns,
		PositionColumns: "e.id, e.event_type, e.aggregate_type, e.aggregate_id, e.version, e.schema_version, e.payload, e.payload_encoding, e.metadata, e.occurred_at",
		TimestampColumn: "occurred_at",

		Scan: s.scanEvents,
	}
}

func (s *SQLEventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	return s.eventJournalReader().ReadAll(ctx)
}

func (s *SQLEventStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	afterID := ""
	if !afterEventID.IsZero() {
		afterID = afterEventID.String()
	}

	return s.eventJournalReader().ReadFrom(ctx, afterID, limit)
}
