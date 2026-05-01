package storage

import (
	"context"
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

	return NewSQLEventStore(db), mock
}

func testEvent(t *testing.T, version int, opts ...event.Option) *event.Core {
	t.Helper()

	return testEventWithAggID(t, id.NewAggregateID(), version, opts...)
}

func testEventWithAggID(t *testing.T, aggID id.AggregateID, version int, opts ...event.Option) *event.Core {
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
	evt := testEvent(t, 1)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`,
	)).WithArgs("User", evt.AggregateID()).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
	)).WithArgs(
		evt.ID(),
		"UserCreated",
		"User",
		evt.AggregateID(),
		1,
		evt.Payload(),
		sqlmock.AnyArg(),
		evt.OccurredAt(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

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

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLEventStore_Save_ConcurrencyConflict(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t, 1)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COALESCE(MAX(version), 0) FROM events WHERE aggregate_type = $1 AND aggregate_id = $2`,
	)).WithArgs("User", evt.AggregateID()).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(5))
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

	if err := mock.ExpectationsWereMet(); err != nil {
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
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
	)).WithArgs(
		evt1.ID(), "UserCreated", "User", evt1.AggregateID(), 1, evt1.Payload(), sqlmock.AnyArg(), evt1.OccurredAt(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO events (id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
	)).WithArgs(
		evt2.ID(), "UserCreated", "User", evt2.AggregateID(), 2, evt2.Payload(), sqlmock.AnyArg(), evt2.OccurredAt(),
	).WillReturnResult(sqlmock.NewResult(2, 1))
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

	if err := mock.ExpectationsWereMet(); err != nil {
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

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC`,
	)).WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "event_type", "aggregate_type", "aggregate_id", "version", "payload", "metadata", "occurred_at"},
		).AddRow(
			eventID.String(), "UserCreated", "User", aggID.String(), 1, []byte(`{"name":"test"}`), nil, ts,
		))

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

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLEventStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC`,
	)).WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "event_type", "aggregate_type", "aggregate_id", "version", "payload", "metadata", "occurred_at"},
		))

	events, err := store.Load(context.Background(), "User", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestSQLEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	eventID := id.NewEventID()
	ts := time.Now().Truncate(time.Microsecond)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2 AND version > $3
		ORDER BY version ASC`,
	)).WithArgs("User", aggID, 2).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "event_type", "aggregate_type", "aggregate_id", "version", "payload", "metadata", "occurred_at"},
		).AddRow(
			eventID.String(), "UserUpdated", "User", aggID.String(), 3, []byte(`{"name":"updated"}`), nil, ts,
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

	store, mock := newTestStore(t)
	mock.ExpectClose()

	if err := store.Close(); err != nil {
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

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = $1 AND aggregate_id = $2
		ORDER BY version ASC`,
	)).WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "event_type", "aggregate_type", "aggregate_id", "version", "payload", "metadata", "occurred_at"},
		).AddRow(
			eventID.String(), "UserCreated", "User", aggID.String(), 1, []byte(`{"name":"test"}`), metaJSON, ts,
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
