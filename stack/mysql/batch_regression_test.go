package mysql_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"
)

// TestEventSinkSave_LargeBatchWithinPacketLimit proves dialect-aware
// multi-VALUES batching cannot exceed max_allowed_packet: 2000 events x 8 KiB
// payloads total ~16 MiB, which one 3276-row statement would exceed on a
// default MariaDB (16 MiB packet). The statement byte cap must split the Save
// into packet-safe chunks that all land and load back exactly once.
func TestEventSinkSave_LargeBatchWithinPacketLimit(t *testing.T) {
	dsn := mysqlDSN(t)

	b, err := mysql.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	ctx := context.Background()
	ref := id.NewStreamRef("Todo", id.NewStreamID())

	const eventCount = 2000
	payload := make([]byte, 8<<10)
	for i := range payload {
		payload[i] = byte('a' + i%26)
	}

	events := make([]event.Event, eventCount)
	for i := range events {
		evt, err := event.NewEvent(
			event.Type("doc.attached"),
			ref.ID,
			id.StreamType("Todo"),
			event.Version(i+1),
			payload,
		)
		if err != nil {
			t.Fatalf("NewEvent %d: %v", i, err)
		}
		events[i] = evt
	}

	if err := b.EventSink.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("EventSink.Save: %v", err)
	}

	loaded, err := b.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("EventSource.Load: %v", err)
	}

	if len(loaded) != eventCount {
		t.Fatalf("loaded %d events, want %d", len(loaded), eventCount)
	}

	if got := loaded[eventCount-1].Version(); got != event.Version(eventCount) {
		t.Fatalf("last version = %d, want %d", got, eventCount)
	}
}
