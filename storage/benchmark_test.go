package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

func mockEventRows(aggID id.AggregateID, now time.Time, payload, metaJSON []byte) *sqlmock.Rows {
	return sqlmock.NewRows(eventColumns()).
		AddRow(
			id.NewEventID().String(), "user.created", "User",
			aggID.String(), 1, 1, payload, "json", metaJSON, now,
		)
}

func benchTestEvent(b *testing.B) (*event.ImmutableEvent, id.AggregateID) {
	b.Helper()
	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		event.Type("user.created"), aggID, event.AggregateType("User"),
		event.Version(1),
		[]byte(`{"name":"test","email":"test@example.com"}`),
		event.WithEventID(id.NewEventID()),
		event.WithCorrelationID(id.NewCorrelationID()),
	)
	if err != nil {
		b.Fatalf("NewEvent: %v", err)
	}
	return evt, aggID
}

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
	payload, err := json.Marshal(map[string]string{"name": "test"})
	if err != nil {
		b.Fatalf("marshal payload: %v", err)
	}

	metaJSON, err := json.Marshal(map[string]string{})
	if err != nil {
		b.Fatalf("marshal meta: %v", err)
	}
	for b.Loop() {
		rows := mockEventRows(aggID, now, payload, metaJSON)

		mock.ExpectQuery("SELECT (.+) FROM events WHERE aggregate_type").
			WithArgs("User", aggID.String()).
			WillReturnRows(rows)

		_, _ = store.Load(
			context.Background(),
			event.NewAggregateRef(event.AggregateType("User"), aggID),
		)

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
				1, 1, payload, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(0, 1))

		_ = store.Save(
			context.Background(), event.NewAggregateRef(event.AggregateType("User"), aggID),
			[]event.Event{evt}, event.Version(1),
		)

		if err := mock.ExpectationsWereMet(); err != nil {
			b.Fatalf("unmet expectations: %v", err)
		}
	}
}

func BenchmarkSQLEventStore_LoadToVersion(b *testing.B) {
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
	payload, err := json.Marshal(map[string]string{"name": "test"})
	if err != nil {
		b.Fatal(err)
	}
	metaJSON, err := json.Marshal(map[string]string{})
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		rows := mockEventRows(aggID, now, payload, metaJSON)

		mock.ExpectQuery("SELECT (.+) FROM events WHERE aggregate_type").
			WithArgs("User", aggID.String(), 2).
			WillReturnRows(rows)

		_, _ = store.LoadToVersion(
			context.Background(),
			event.NewAggregateRef(event.AggregateType("User"), aggID),
			2,
		)

		if err := mock.ExpectationsWereMet(); err != nil {
			b.Fatalf("unmet expectations: %v", err)
		}
	}
}
