package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func BenchmarkSQLEventStore_Load(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatalf("create sqlmock: %v", err)
	}

	store, err := NewSQLEventStore(db)
	if err != nil {
		b.Fatalf("NewSQLEventStore: %v", err)
	}

	aggID := id.NewAggregateID()
	now := time.Now()
	payload, _ := json.Marshal(map[string]string{"name": "test"})
	metaJSON, _ := json.Marshal(map[string]string{})
	columns := []string{
		"id", "event_type", "aggregate_type", "aggregate_id",
		"version", "schema_version", "payload", "metadata", "occurred_at",
	}

	for b.Loop() {
		rows := sqlmock.NewRows(columns).
			AddRow(
				id.NewEventID().String(), "user.created", "User",
				aggID.String(), 1, 1, payload, metaJSON, now,
			)

		mock.ExpectQuery("SELECT (.+) FROM events WHERE aggregate_type").
			WithArgs("User", aggID.String()).
			WillReturnRows(rows)

		_, _ = store.Load(context.Background(), "User", aggID)

		if err := mock.ExpectationsWereMet(); err != nil {
			b.Fatalf("unmet expectations: %v", err)
		}
	}
}

func BenchmarkSQLEventStore_Save(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatalf("create sqlmock: %v", err)
	}

	store, err := NewSQLEventStore(db)
	if err != nil {
		b.Fatalf("NewSQLEventStore: %v", err)
	}

	aggID := id.NewAggregateID()
	payload := []byte(`{"name":"test"}`)

	for b.Loop() {
		evt, _ := event.NewEvent(
			event.Type("user.created"), aggID, event.AggregateType("User"),
			event.Version(1), payload,
		)

		mock.ExpectExec("INSERT INTO events").
			WithArgs(
				sqlmock.AnyArg(), "user.created", "User", aggID.String(),
				1, 1, payload, sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(0, 1))

		_ = store.Save(
			context.Background(), event.AggregateType("User"),
			aggID, []event.Event{evt}, event.Version(1),
		)

		if err := mock.ExpectationsWereMet(); err != nil {
			b.Fatalf("unmet expectations: %v", err)
		}
	}
}

func BenchmarkPebbleSerialize(b *testing.B) {
	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(
		event.Type("user.created"), aggID, event.AggregateType("User"),
		event.Version(1),
		[]byte(`{"name":"test","email":"test@example.com"}`),
		event.WithEventID(id.NewEventID()),
		event.WithCorrelationID(id.NewCorrelationID()),
	)

	adapter := &CQRSAdapter{prefix: "test"}

	for b.Loop() {
		_, err := adapter.serializeEvent(evt)
		if err != nil {
			b.Fatalf("serializeEvent: %v", err)
		}
	}
}

func BenchmarkPebbleDeserialize(b *testing.B) {
	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(
		event.Type("user.created"), aggID, event.AggregateType("User"),
		event.Version(1),
		[]byte(`{"name":"test","email":"test@example.com"}`),
		event.WithEventID(id.NewEventID()),
		event.WithCorrelationID(id.NewCorrelationID()),
	)

	adapter := &CQRSAdapter{prefix: "test"}
	data, _ := adapter.serializeEvent(evt)

	for b.Loop() {
		_, err := adapter.deserializeEvent(data)
		if err != nil {
			b.Fatalf("deserializeEvent: %v", err)
		}
	}
}
