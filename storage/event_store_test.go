package storage

import (
	"context"
	"database/sql/driver"
	"errors"
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
		"version", "payload", "metadata", "occurred_at",
	}
}

const loadQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC`

const loadFromVersionQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version > $3
		ORDER BY version ASC`

const insertQuery = `INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

const versionQuery = `SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`

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

func expectSaveSuccess(mock sqlmock.Sqlmock, evt *event.Core) {
	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 0)
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).WithArgs(
		evt.ID(),
		"UserCreated", "User", evt.AggregateID(), 1, evt.Payload(), sqlmock.AnyArg(), evt.OccurredAt(),
	).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
}

func testEvent(t *testing.T) *event.Core {
	t.Helper()

	return testEventWithAggID(t, id.NewAggregateID(), 1)
}

func testEventWithAggID(
	t *testing.T,
	aggID id.AggregateID,
	version int,
	opts ...event.Option,
) *event.Core {
	t.Helper()

	evt, err := event.NewEvent(
		"UserCreated",
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

func TestSQLEventStore_Save_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	expectSaveSuccess(mock, evt)

	err := store.Save(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLEventStore_Save_ConcurrencyConflict(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 5)
	mock.ExpectRollback()

	err := store.Save(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
		event.Version(0),
	)
	if err == nil {
		t.Fatal("expected concurrency conflict error")
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Errorf("error should wrap event.ErrVersionConflict, got: %v", err)
	}

	if !errors.Is(err, ErrConcurrencyConflict) {
		t.Errorf("error should wrap ErrConcurrencyConflict, got: %v", err)
	}

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLEventStore_Save_EmptyEvents(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	aggID := id.NewAggregateID()

	err := store.Save(context.Background(), "User", aggID, nil, event.Version(0))
	if err != nil {
		t.Fatalf("Save with empty events: %v", err)
	}
}

func TestSQLEventStore_AppendBatch_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	evt1 := testEventWithAggID(t, aggID, 1)
	evt2 := testEventWithAggID(t, aggID, 2)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).WithArgs(
		evt1.ID(),
		"UserCreated", "User", evt1.AggregateID(), 1, evt1.Payload(), sqlmock.AnyArg(), evt1.OccurredAt(),
	).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).WithArgs(
		evt2.ID(),
		"UserCreated", "User", evt2.AggregateID(), 2, evt2.Payload(), sqlmock.AnyArg(), evt2.OccurredAt(),
	).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	err := store.AppendBatch(
		context.Background(),
		"User",
		evt1.AggregateID(),
		[]event.Event{evt1, evt2},
	)
	if err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLEventStore_AppendBatch_EmptyEvents(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	aggID := id.NewAggregateID()

	err := store.AppendBatch(context.Background(), "User", aggID, nil)
	if err != nil {
		t.Fatalf("AppendBatch with empty events: %v", err)
	}
}

func TestSQLEventStore_Load_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	eventID := id.NewEventID()
	ts := time.Now().Truncate(time.Microsecond)

	expectLoadRows(
		mock,
		aggID,
		eventID.String(),
		"UserCreated",
		"User",
		aggID.String(),
		1,
		[]byte(`{"name":"test"}`),
		nil,
		ts,
	)

	events, err := store.Load(context.Background(), "User", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type() != "UserCreated" {
		t.Errorf("Type = %q, want UserCreated", events[0].Type())
	}

	if events[0].ID() != eventID {
		t.Errorf("ID = %v, want %v", events[0].ID(), eventID)
	}

	if !events[0].OccurredAt().Equal(ts) {
		t.Errorf("OccurredAt = %v, want %v", events[0].OccurredAt(), ts)
	}

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLEventStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	expectLoadEmpty(mock, aggID)

	events, err := store.Load(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected ErrAggregateNotFound for empty result, got nil")
	}

	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Errorf("expected ErrAggregateNotFound, got %v", err)
	}

	if events != nil {
		t.Fatalf("expected nil events, got %d", len(events))
	}
}

func TestSQLEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	eventID := id.NewEventID()
	ts := time.Now().Truncate(time.Microsecond)

	mock.ExpectQuery(regexp.QuoteMeta(loadFromVersionQuery)).
		WithArgs("User", aggID, 2).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			eventID.String(),
			"UserUpdated", "User", aggID.String(), 3, []byte(`{"name":"updated"}`), nil, ts,
		))

	events, err := store.LoadFromVersion(context.Background(), "User", aggID, event.Version(2))
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Version() != 3 {
		t.Errorf("Version = %d, want 3", events[0].Version())
	}
}

func TestSQLEventStore_Delete(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`,
	)).WithArgs("User", aggID).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := store.Delete(context.Background(), "User", aggID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestSQLEventStore_Close(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)

	err := store.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestMarshalMetadata_Nil(t *testing.T) {
	result, err := marshalMetadata(nil)
	if err != nil {
		t.Fatalf("marshalMetadata(nil): %v", err)
	}

	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSQLEventStore_Save_BeginTxFailure(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := store.Save(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
		event.Version(0),
	)
	if err == nil {
		t.Fatal("expected error for BeginTx failure")
	}
}

func TestSQLEventStore_Save_VersionQueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(versionQuery)).
		WithArgs("User", evt.AggregateID()).
		WillReturnError(errors.New("query failed"))
	mock.ExpectRollback()

	err := store.Save(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
		event.Version(0),
	)
	if err == nil {
		t.Fatal("expected error for version query failure")
	}
}

func TestSQLEventStore_Save_InsertError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 0)
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).WithArgs(
		evt.ID(),
		"UserCreated",
		"User",
		evt.AggregateID(),
		1,
		evt.Payload(),
		sqlmock.AnyArg(),
		evt.OccurredAt(),
	).WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	err := store.Save(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
		event.Version(0),
	)
	if err == nil {
		t.Fatal("expected error for insert failure")
	}
}

func TestSQLEventStore_Save_CommitError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 0)
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).WithArgs(
		evt.ID(),
		"UserCreated",
		"User",
		evt.AggregateID(),
		1,
		evt.Payload(),
		sqlmock.AnyArg(),
		evt.OccurredAt(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err := store.Save(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
		event.Version(0),
	)
	if err == nil {
		t.Fatal("expected error for commit failure")
	}
}

func TestSQLEventStore_AppendBatch_BeginTxFailure(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := store.AppendBatch(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
	)
	if err == nil {
		t.Fatal("expected error for BeginTx failure")
	}
}

func TestSQLEventStore_AppendBatch_InsertError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	err := store.AppendBatch(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
	)
	if err == nil {
		t.Fatal("expected error for insert failure")
	}
}

func TestSQLEventStore_AppendBatch_CommitError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).WithArgs(
		evt.ID(),
		"UserCreated",
		"User",
		evt.AggregateID(),
		1,
		evt.Payload(),
		sqlmock.AnyArg(),
		evt.OccurredAt(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err := store.AppendBatch(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
	)
	if err == nil {
		t.Fatal("expected error for commit failure")
	}
}

func TestSQLEventStore_Load_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnError(errors.New("query failed"))

	_, err := store.Load(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected error for query failure")
	}
}

func TestSQLEventStore_LoadFromVersion_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadFromVersionQuery)).
		WithArgs("User", aggID, 2).
		WillReturnError(errors.New("query failed"))

	_, err := store.LoadFromVersion(context.Background(), "User", aggID, event.Version(2))
	if err == nil {
		t.Fatal("expected error for query failure")
	}
}

func TestScanEvents_InvalidAggregateID(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", "bad").
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			"valid-id", "UserCreated", "User", "not-a-valid-ulid", 1, nil, nil, time.Now(),
		))

	_, err := store.Load(context.Background(), "User", id.NewAggregateID())
	if err == nil {
		t.Fatal("expected error for invalid aggregate ID")
	}
}

func TestScanEvents_InvalidEventID(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			"not-a-valid-ulid", "UserCreated", "User", aggID.String(), 1, nil, nil, time.Now(),
		))

	_, err := store.Load(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected error for invalid event ID")
	}
}

func TestScanEvents_RowScanError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", "bad").
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "event_type"},
		).AddRow("only-two-columns", "UserCreated"))

	_, err := store.Load(context.Background(), "User", id.NewAggregateID())
	if err == nil {
		t.Fatal("expected error for row scan failure")
	}
}

func TestScanEvents_InvalidMetadata(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	eventID := id.NewEventID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()).
			AddRow(
				eventID.String(),
				"UserCreated", "User", aggID.String(), 1, nil, []byte(`{invalid`), time.Now(),
			))

	_, err := store.Load(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected error for invalid metadata JSON")
	}
}

func TestSQLEventStore_Delete_Error(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`,
	)).WithArgs("User", aggID).
		WillReturnError(errors.New("delete failed"))

	err := store.Delete(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected error for delete failure")
	}
}

func TestSchema_ContainsExpectedDDL(t *testing.T) {
	ddl := Schema()

	if !regexp.MustCompile(`(?s)CREATE TABLE.*events`).MatchString(ddl) {
		t.Error("Schema() missing CREATE TABLE events")
	}

	if !regexp.MustCompile(`UNIQUE\(aggregate_type,\s*aggregate_id,\s*version\)`).MatchString(ddl) {
		t.Error("Schema() missing UNIQUE constraint")
	}

	if !regexp.MustCompile(`idx_events_aggregate`).MatchString(ddl) {
		t.Error("Schema() missing idx_events_aggregate index")
	}

	if !regexp.MustCompile(`idx_events_type`).MatchString(ddl) {
		t.Error("Schema() missing idx_events_type index")
	}
}

func TestSQLEventStore_SQLInjectionSafety(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)

	maliciousAggType := event.AggregateType("User'; DROP TABLE events; --")
	maliciousAggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs(string(maliciousAggType), maliciousAggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()))

	events, err := store.Load(context.Background(), maliciousAggType, maliciousAggID)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("Load with malicious input: expected ErrAggregateNotFound, got %v", err)
	}

	if events != nil {
		t.Errorf("expected nil events, got %d", len(events))
	}
}

func TestMarshalMetadata_Full(t *testing.T) {
	meta := event.NewMetadata()
	meta.CorrelationID = id.NewCorrelationID()
	meta.UserID = id.NewUserID()
	meta.Custom = map[event.MetadataKey]string{"env": "test"}

	result, err := marshalMetadata(meta)
	if err != nil {
		t.Fatalf("marshalMetadata: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}

func TestScanEvents_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	eventID := id.NewEventID()
	ts := time.Now().Truncate(time.Microsecond)
	cid := id.NewCorrelationID()
	uid := id.NewUserID()

	evt, err := event.NewEvent(
		"UserCreated",
		aggID,
		"User",
		1,
		[]byte(`{"name":"test"}`),
		event.WithEventID(eventID),
		event.WithOccurredAt(ts),
		event.WithCorrelationID(cid),
		event.WithUserID(uid),
		event.WithCustom("env", "test"),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	metaJSON, err := marshalMetadata(evt.Metadata())
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()).
			AddRow(
				eventID.String(),
				"UserCreated", "User", aggID.String(), 1, []byte(`{"name":"test"}`), metaJSON, ts,
			))

	loaded, err := store.Load(context.Background(), "User", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	got := loaded[0]
	if got.ID() != eventID {
		t.Errorf("ID = %v, want %v", got.ID(), eventID)
	}

	if !got.OccurredAt().Equal(ts) {
		t.Errorf("OccurredAt = %v, want %v", got.OccurredAt(), ts)
	}

	if got.Metadata() == nil {
		t.Fatal("Metadata is nil")
	}

	if got.Metadata().CorrelationID != cid {
		t.Errorf("CorrelationID = %v, want %v", got.Metadata().CorrelationID, cid)
	}

	if got.Metadata().UserID != uid {
		t.Errorf("UserID = %v, want %v", got.Metadata().UserID, uid)
	}

	if got.Metadata().Custom["env"] != "test" {
		t.Errorf("Custom[env] = %q, want %q", got.Metadata().Custom["env"], "test")
	}
}

const loadAllQuery = `SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		ORDER BY occurred_at ASC`

func TestSQLEventStore_LoadAll_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()
	eventID1 := id.NewEventID()
	eventID2 := id.NewEventID()
	ts1 := time.Now().Truncate(time.Microsecond)
	ts2 := ts1.Add(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(loadAllQuery)).
		WillReturnRows(
			sqlmock.NewRows(eventColumns()).
				AddRow(eventID1.String(), "UserCreated", "User", aggID1.String(), 1, []byte(`{}`), nil, ts1).
				AddRow(eventID2.String(), "UserCreated", "User", aggID2.String(), 1, []byte(`{}`), nil, ts2),
		)

	events, err := store.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].ID() != eventID1 {
		t.Errorf("events[0].ID = %v, want %v", events[0].ID(), eventID1)
	}

	if events[1].ID() != eventID2 {
		t.Errorf("events[1].ID = %v, want %v", events[1].ID(), eventID2)
	}

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLEventStore_LoadAll_Empty(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(loadAllQuery)).
		WillReturnRows(sqlmock.NewRows(eventColumns()))

	events, err := store.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestSQLEventStore_LoadAll_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(loadAllQuery)).
		WillReturnError(errors.New("query failed"))

	_, err := store.LoadAll(context.Background())
	if err == nil {
		t.Fatal("expected error for query failure")
	}
}
