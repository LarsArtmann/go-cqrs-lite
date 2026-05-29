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
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func testLoadAllSuccess(
	t *testing.T,
	store *SQLEventStore,
	mock sqlmock.Sqlmock,
	fn func() ([]event.Event, error),
	eventID1, eventID2 id.EventID,
) {
	events, err := fn()
	if err != nil {
		t.Fatalf("LoadAll/ReadAll: %v", err)
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

func setupLoadAllSuccess(
	t *testing.T,
	store *SQLEventStore,
	mock sqlmock.Sqlmock,
) (id.EventID, id.EventID) {
	t.Helper()

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

	return eventID1, eventID2
}

func TestSQLEventStore_LoadAll_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	eventID1, eventID2 := setupLoadAllSuccess(t, store, mock)

	testLoadAllSuccess(t, store, mock, func() ([]event.Event, error) {
		return store.LoadAll(context.Background())
	}, eventID1, eventID2)
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
	eventID1, eventID2 := setupLoadAllSuccess(t, store, mock)

	testLoadAllSuccess(t, store, mock, func() ([]event.Event, error) {
		return store.ReadAll(context.Background())
	}, eventID1, eventID2)
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

	backwardsLoader := event.BackwardsSource(store)
	events, err := backwardsLoader.LoadBackwards(context.Background(), "User", aggID)
	if err != nil {
		t.Fatalf("LoadBackwards: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	testhelpers.AssertEventType(t, events, 0, "UserUpdated")
	testhelpers.AssertEventType(t, events, 1, "UserCreated")
}

func TestSQLEventStore_LoadBackwards_NotFound(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()

	mock.ExpectQuery(regexp.QuoteMeta(loadBackwardsQuery)).
		WithArgs("User", aggID).
		WillReturnRows(sqlmock.NewRows(eventColumns()))

	backwardsLoader := event.BackwardsSource(store)
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
