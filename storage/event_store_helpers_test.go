package storage

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newTestStore(t *testing.T) (*SQLEventStore, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	store, err := NewSQLEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}

	return store, mock
}

func eventColumns() []string {
	return []string{
		"id", "event_type", "aggregate_type", "aggregate_id",
		"version", "schema_version", "payload", "metadata", "occurred_at",
	}
}

const loadQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC`

const loadFromVersionQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version > $3
		ORDER BY version ASC`

const insertQuery = `INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

const versionQuery = `SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`

const loadAllQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		ORDER BY occurred_at ASC`

const loadBackwardsQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version DESC`

func expectLoadRows(mock sqlmock.Sqlmock, aggID id.AggregateID, rows ...driver.Value) {
	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(rows...))
}

func expectLoadEmpty(mock sqlmock.Sqlmock, aggID id.AggregateID) {
	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()))
}

func expectVersionCheck(mock sqlmock.Sqlmock, aggID id.AggregateID, version int) {
	mock.ExpectQuery(regexp.QuoteMeta(versionQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(version))
}

func expectSaveSuccess(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) {
	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 0)
	expectInsertSuccess(mock, evt)
	mock.ExpectCommit()
}

func expectInsertSuccess(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) {
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).WithArgs(
		evt.ID(),
		"UserCreated", "User", evt.AggregateID(), 1, evt.SchemaVersion().Int(), evt.Payload(), sqlmock.AnyArg(), evt.OccurredAt(),
	).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func saveEvt(t *testing.T, store *SQLEventStore, evt *event.ImmutableEvent) error {
	t.Helper()

	return store.Save(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
		event.Version(0),
	)
}

func appendBatchEvt(t *testing.T, store *SQLEventStore, evt *event.ImmutableEvent) error {
	t.Helper()

	return store.AppendBatch(context.Background(), "User", evt.AggregateID(), []event.Event{evt})
}

func testEvent(t *testing.T) *event.ImmutableEvent {
	t.Helper()

	return testEventWithAggID(t, "UserCreated", id.NewAggregateID(), 1)
}

func testEventWithAggID(
	t *testing.T,
	eventType event.Type,
	aggID id.AggregateID,
	version event.Version,
	opts ...event.Option,
) *event.ImmutableEvent {
	t.Helper()

	evt, err := event.NewEvent(
		eventType,
		aggID,
		"User",
		version,
		[]byte(`{"name":"test"}`),
		opts...,
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	return evt
}

func testEventWithTimestamp(t *testing.T, eventType event.Type, aggID id.AggregateID, version event.Version, ts time.Time) *event.ImmutableEvent {
	t.Helper()

	return testEventWithAggID(t, eventType, aggID, version, event.WithOccurredAt(ts))
}
