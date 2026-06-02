package storage

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

const loadToVersionQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version <= $3 ORDER BY version ASC`

const loadToTimestampQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND occurred_at <= $3 ORDER BY version ASC`

const loadAllFromPositionQuery = `SELECT e.id, e.event_type, e.aggregate_type, e.aggregate_id, e.version, e.schema_version, e.payload, e.payload_encoding, e.metadata, e.occurred_at
		FROM events e
		JOIN events c ON c.id = $1
		WHERE (e.occurred_at > c.occurred_at) OR (e.occurred_at = c.occurred_at AND e.id > $2)
		ORDER BY e.occurred_at ASC, e.id ASC LIMIT $3`

func mockEventRowsForTest(evt *event.ImmutableEvent, aggID id.AggregateID) *sqlmock.Rows {
	return sqlmock.NewRows(eventColumns()).AddRow(
		evt.ID(), "UserCreated", "User", aggID,
		1, 1, evt.Payload(), "json", nil, evt.OccurredAt(),
	)
}

func TestSQLEventStore_LoadToVersion_Mock_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	evt := testEventWithAggID(t, "UserCreated", aggID, 1)

	mock.ExpectQuery(regexp.QuoteMeta(loadToVersionQuery)).
		WithArgs("User", aggID, 1).
		WillReturnRows(mockEventRowsForTest(evt, aggID))

	events, err := store.LoadToVersion(
		context.Background(),
		event.NewAggregateRef("User", aggID),
		1,
	)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestSQLEventStore_LoadToVersion_Mock_NotFound(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadToVersionQuery)).
		WithArgs("User", aggID, 5).
		WillReturnRows(sqlmock.NewRows(eventColumns()))

	_, err := store.LoadToVersion(context.Background(), event.NewAggregateRef("User", aggID), 5)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestSQLEventStore_LoadToVersion_Mock_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadToVersionQuery)).
		WithArgs("User", aggID, 5).
		WillReturnError(errors.New("connection lost"))

	_, err := store.LoadToVersion(context.Background(), event.NewAggregateRef("User", aggID), 5)
	if err == nil {
		t.Fatal("expected error from query failure")
	}
}

func TestSQLEventStore_LoadToTimestamp_Mock_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	evt := testEventWithAggID(t, "UserCreated", aggID, 1)

	mock.ExpectQuery(regexp.QuoteMeta(loadToTimestampQuery)).
		WithArgs("User", aggID, evt.OccurredAt()).
		WillReturnRows(mockEventRowsForTest(evt, aggID))

	events, err := store.LoadToTimestamp(
		context.Background(),
		event.NewAggregateRef("User", aggID),
		evt.OccurredAt(),
	)
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestSQLEventStore_LoadToTimestamp_Mock_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadToTimestampQuery)).
		WithArgs("User", aggID, sqlmock.AnyArg()).
		WillReturnError(errors.New("connection lost"))

	_, err := store.LoadToTimestamp(
		context.Background(),
		event.NewAggregateRef("User", aggID),
		testEvent(t).OccurredAt(),
	)
	if err == nil {
		t.Fatal("expected error from query failure")
	}
}

func mockLoadAllFromPosition(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) {
	mock.ExpectQuery(regexp.QuoteMeta(loadAllFromPositionQuery)).
		WithArgs(evt.ID().String(), evt.ID().String(), 10).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			evt.ID(), "UserCreated", "User", evt.AggregateID(),
			1, 1, evt.Payload(), "json", nil, evt.OccurredAt(),
		))
}

func TestSQLEventStore_Load_Mock_ScanError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			"invalid-id", "", "User", aggID, 1, 1, nil, "", nil, testEvent(t).OccurredAt(),
		))

	_, err := store.Load(context.Background(), event.NewAggregateRef("User", aggID))
	if err == nil {
		t.Fatal("expected error from scan with invalid event ID")
	}
}

func TestSQLEventStore_ReadFrom_Mock_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	evt := testEventWithAggID(t, "UserCreated", aggID, 1)

	mockLoadAllFromPosition(mock, evt)

	events, err := store.ReadFrom(context.Background(), evt.ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestSQLEventStore_ReadFrom_Mock_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evtID := id.NewEventID()

	mock.ExpectQuery(regexp.QuoteMeta(loadAllFromPositionQuery)).
		WithArgs(evtID.String(), evtID.String(), 10).
		WillReturnError(errors.New("connection lost"))

	_, err := store.ReadFrom(context.Background(), evtID, 10)
	if err == nil {
		t.Fatal("expected error from query failure")
	}
}

func TestSQLEventStore_ReadAll_Mock_ScanError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	mockLoadAllScanError(mock, t)

	_, err := store.ReadAll(context.Background())
	expectScanError(t, err, "ReadAll")
}

func mockLoadAllScanError(mock sqlmock.Sqlmock, t *testing.T) {
	aggID := id.NewAggregateID()
	mock.ExpectQuery(regexp.QuoteMeta(loadAllQuery)).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			"invalid-id", "", "User", aggID, 1, 1, nil, "", nil, testEvent(t).OccurredAt(),
		))
}

func expectScanError(t *testing.T, err error, method string) {
	if err == nil {
		t.Fatalf("expected error from scan with invalid event ID in %s", method)
	}
}
