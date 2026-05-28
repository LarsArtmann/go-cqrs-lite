package storage

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

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
			"UserUpdated", "User", aggID.String(), 3, 1, []byte(`{"name":"updated"}`), nil, ts,
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

func TestScanEvents_InvalidAggregateID(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", "bad").
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			"valid-id", "UserCreated", "User", "not-a-valid-ulid", 1, 1, nil, nil, time.Now(),
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
			"not-a-valid-ulid", "UserCreated", "User", aggID.String(), 1, 1, nil, nil, time.Now(),
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
				"UserCreated", "User", aggID.String(), 1, 1, nil, []byte(`{invalid`), time.Now(),
			))

	_, err := store.Load(context.Background(), "User", aggID)
	if err == nil {
		t.Fatal("expected error for invalid metadata JSON")
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
		WillReturnRows(
			sqlmock.NewRows(eventColumns()).
				AddRow(
					eventID.String(),
					"UserCreated", "User", aggID.String(), 1, 1, []byte(`{"name":"test"}`), metaJSON, ts,
				),
		)

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
				AddRow(eventID1.String(), "UserCreated", "User", aggID1.String(), 1, 1, []byte(`{}`), nil, ts1).
				AddRow(eventID2.String(), "UserCreated", "User", aggID2.String(), 1, 1, []byte(`{}`), nil, ts2),
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

func TestSQLEventStore_ReadAll_Success(t *testing.T) {
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
				AddRow(eventID1.String(), "UserCreated", "User", aggID1.String(), 1, 1, []byte(`{}`), nil, ts1).
				AddRow(eventID2.String(), "UserCreated", "User", aggID2.String(), 1, 1, []byte(`{}`), nil, ts2),
		)

	events, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
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

func TestSQLEventStore_ReadAll_Empty(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(loadAllQuery)).
		WillReturnRows(sqlmock.NewRows(eventColumns()))

	events, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestSQLEventStore_ReadAll_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)

	mock.ExpectQuery(regexp.QuoteMeta(loadAllQuery)).
		WillReturnError(errors.New("query failed"))

	_, err := store.ReadAll(context.Background())
	if err == nil {
		t.Fatal("expected error for query failure")
	}
}

func TestSQLEventStore_LoadBackwards_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	evtID1 := id.NewEventID()
	evtID2 := id.NewEventID()
	ts := time.Now().Truncate(time.Microsecond)

	mock.ExpectQuery(regexp.QuoteMeta(loadBackwardsQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()).
			AddRow(evtID2.String(), "UserUpdated", "User", aggID.String(), 2, 1, nil, nil, ts).
			AddRow(evtID1.String(), "UserCreated", "User", aggID.String(), 1, 1, nil, nil, ts))

	backwardsLoader := event.BackwardsLoader(store)
	events, err := backwardsLoader.LoadBackwards(context.Background(), "User", aggID)
	if err != nil {
		t.Fatalf("LoadBackwards: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Type() != "UserUpdated" {
		t.Errorf("first event = %q, want UserUpdated", events[0].Type())
	}

	if events[1].Type() != "UserCreated" {
		t.Errorf("second event = %q, want UserCreated", events[1].Type())
	}
}

func TestSQLEventStore_LoadBackwards_NotFound(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadBackwardsQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()))

	backwardsLoader := event.BackwardsLoader(store)
	_, err := backwardsLoader.LoadBackwards(context.Background(), "User", aggID)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got %v", err)
	}
}

func TestSchema_ContainsExpectedDDL(t *testing.T) {
	ddl := Schema()

	if !regexp.MustCompile(`(?s)CREATE TABLE.*events`).MatchString(ddl) {
		t.Error("Schema() missing CREATE TABLE events")
	}

	if !regexp.MustCompile(`(?s)CREATE TABLE.*outbox`).MatchString(ddl) {
		t.Error("Schema() missing CREATE TABLE outbox")
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
