package storage

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

const loadToVersionQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version <= $3 ORDER BY version ASC`

const loadToTimestampQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND occurred_at <= $3 ORDER BY version ASC`

const resolveCursorTimestampQuery = `SELECT occurred_at FROM events WHERE id = $1`

const keysetFromPositionQuery = `SELECT e.id, e.event_type, e.aggregate_type, e.aggregate_id, e.version, e.schema_version, e.payload, e.payload_encoding, e.metadata, e.occurred_at
		FROM events e
		WHERE e.occurred_at >= $1 AND (e.occurred_at > $2 OR e.id > $3)
		ORDER BY e.occurred_at ASC, e.id ASC LIMIT $4`

func mockEventRowsForTest(evt event.Event, streamID id.StreamID) *sqlmock.Rows {
	return sqlmock.NewRows(eventColumns()).AddRow(
		evt.ID(), "UserCreated", "User", streamID,
		1, 1, evt.Payload(), "json", nil, evt.OccurredAt(),
	)
}

func TestSQLEventStore_LoadToVersion_Mock_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	streamID := id.NewStreamID()

	evt := testEventWithAggID(t, "UserCreated", streamID, 1)

	mock.ExpectQuery(regexp.QuoteMeta(loadToVersionQuery)).
		WithArgs("User", streamID, 1).
		WillReturnRows(mockEventRowsForTest(evt, streamID))

	events, err := store.LoadToVersion(
		context.Background(),
		id.NewStreamRef("User", streamID),
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
	streamID := id.NewStreamID()

	mock.ExpectQuery(regexp.QuoteMeta(loadToVersionQuery)).
		WithArgs("User", streamID, 5).
		WillReturnRows(sqlmock.NewRows(eventColumns()))

	_, err := store.LoadToVersion(context.Background(), id.NewStreamRef("User", streamID), 5)
	if !errors.Is(err, event.ErrStreamNotFound) {
		t.Fatalf("expected ErrStreamNotFound, got: %v", err)
	}
}

func TestSQLEventStore_LoadToVersion_Mock_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	streamID := id.NewStreamID()

	mock.ExpectQuery(regexp.QuoteMeta(loadToVersionQuery)).
		WithArgs("User", streamID, 5).
		WillReturnError(errors.New("connection lost"))

	_, err := store.LoadToVersion(context.Background(), id.NewStreamRef("User", streamID), 5)
	if err == nil {
		t.Fatal("expected error from query failure")
	}
}

func TestSQLEventStore_LoadToTimestamp_Mock_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	streamID := id.NewStreamID()

	evt := testEventWithAggID(t, "UserCreated", streamID, 1)

	mock.ExpectQuery(regexp.QuoteMeta(loadToTimestampQuery)).
		WithArgs("User", streamID, evt.OccurredAt()).
		WillReturnRows(mockEventRowsForTest(evt, streamID))

	events, err := store.LoadToTimestamp(
		context.Background(),
		id.NewStreamRef("User", streamID),
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
	streamID := id.NewStreamID()

	mock.ExpectQuery(regexp.QuoteMeta(loadToTimestampQuery)).
		WithArgs("User", streamID, sqlmock.AnyArg()).
		WillReturnError(errors.New("connection lost"))

	_, err := store.LoadToTimestamp(
		context.Background(),
		id.NewStreamRef("User", streamID),
		testEvent(t).OccurredAt(),
	)
	if err == nil {
		t.Fatal("expected error from query failure")
	}
}

func mockLoadAllFromPosition(mock sqlmock.Sqlmock, evt event.Event) {
	mock.ExpectQuery(regexp.QuoteMeta(resolveCursorTimestampQuery)).
		WithArgs(evt.ID().String()).
		WillReturnRows(sqlmock.NewRows([]string{"occurred_at"}).AddRow(evt.OccurredAt()))

	mock.ExpectQuery(regexp.QuoteMeta(keysetFromPositionQuery)).
		WithArgs(evt.OccurredAt(), evt.OccurredAt(), evt.ID().String(), 10).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			evt.ID(), "UserCreated", "User", evt.StreamID(),
			1, 1, evt.Payload(), "json", nil, evt.OccurredAt(),
		))
}

func TestSQLEventStore_Load_Mock_ScanError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	streamID := id.NewStreamID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", streamID).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			"invalid-id", "", "User", streamID, 1, 1, nil, "", nil, testEvent(t).OccurredAt(),
		))

	_, err := store.Load(context.Background(), id.NewStreamRef("User", streamID))
	if err == nil {
		t.Fatal("expected error from scan with invalid event ID")
	}
}

func TestSQLEventStore_ReadFrom_Mock_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	streamID := id.NewStreamID()
	evt := testEventWithAggID(t, "UserCreated", streamID, 1)

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
	cursorTS := testEventWithAggID(t, "UserCreated", id.NewStreamID(), 1).OccurredAt()

	mock.ExpectQuery(regexp.QuoteMeta(resolveCursorTimestampQuery)).
		WithArgs(evtID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"occurred_at"}).AddRow(cursorTS))

	mock.ExpectQuery(regexp.QuoteMeta(keysetFromPositionQuery)).
		WithArgs(cursorTS, cursorTS, evtID.String(), 10).
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
	streamID := id.NewStreamID()
	mock.ExpectQuery(regexp.QuoteMeta(loadAllQuery)).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			"invalid-id", "", "User", streamID, 1, 1, nil, "", nil, testEvent(t).OccurredAt(),
		))
}

func expectScanError(t *testing.T, err error, method string) {
	if err == nil {
		t.Fatalf("expected error from scan with invalid event ID in %s", method)
	}
}
