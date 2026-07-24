package storage

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func TestSQLEventStore_Load_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewStreamID()
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
		"json",
		nil,
		ts,
	)

	events, err := store.Load(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), aggID),
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	eventtest.AssertEventType(t, events, 0, "UserCreated")

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
	aggID := id.NewStreamID()

	expectLoadEmpty(mock, aggID)

	events, err := store.Load(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), aggID),
	)
	if err == nil {
		t.Fatal("expected ErrStreamNotFound for empty result, got nil")
	}

	if !errors.Is(err, event.ErrStreamNotFound) {
		t.Errorf("expected ErrStreamNotFound, got %v", err)
	}

	if events != nil {
		t.Fatalf("expected nil events, got %d", len(events))
	}
}

func TestSQLEventStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewStreamID()
	eventID := id.NewEventID()
	ts := time.Now().Truncate(time.Microsecond)

	mock.ExpectQuery(regexp.QuoteMeta(loadFromVersionQuery)).
		WithArgs("User", aggID, 2).
		WillReturnRows(sqlmock.NewRows(eventColumns()).AddRow(
			eventID.String(),
			"UserUpdated", "User", aggID.String(), 3, 1, []byte(`{"name":"updated"}`), "", nil, ts,
		))

	events, err := store.LoadFromVersion(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), aggID),
		event.Version(2),
	)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	eventtest.AssertEventVersion(t, events, 0, 3)
}

func TestSQLEventStore_Load_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewStreamID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnError(errors.New("query failed"))

	_, err := store.Load(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), aggID),
	)
	if err == nil {
		t.Fatal("expected error for query failure")
	}
}

func TestSQLEventStore_LoadFromVersion_QueryError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewStreamID()

	mock.ExpectQuery(regexp.QuoteMeta(loadFromVersionQuery)).
		WithArgs("User", aggID, 2).
		WillReturnError(errors.New("query failed"))

	_, err := store.LoadFromVersion(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), aggID),
		event.Version(2),
	)
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

func TestMarshalMetadata_Empty(t *testing.T) {
	result, err := sqlpkg.MarshalMetadata(event.Metadata{})
	if err != nil {
		t.Fatalf("sqlpkg.MarshalMetadata(event.Metadata{}): %v", err)
	}

	if len(result) == 0 {
		t.Errorf("expected non-empty result for empty metadata, got %v", result)
	}
}

func TestMarshalMetadata_Full(t *testing.T) {
	meta := event.NewMetadata()
	meta.CorrelationID = id.NewCorrelationID()
	meta.UserID = id.NewUserID()
	meta.Custom = map[event.MetadataKey]string{"env": "test"}

	result, err := sqlpkg.MarshalMetadata(meta)
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
			"valid-id", "UserCreated", "User", "not-a-valid-ulid", 1, 1, nil, "", nil, time.Now(),
		))

	_, err := store.Load(context.Background(), id.NewStreamRef("User", id.NewStreamID()))
	if err == nil {
		t.Fatal("expected error for invalid stream ID")
	}
}

func TestScanEvents_InvalidEventID(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewStreamID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(
			sqlmock.NewRows(eventColumns()).AddRow(
				"not-a-valid-ulid", "UserCreated", "User", aggID.String(), 1, 1, nil, "", nil, time.Now(),
			),
		)

	_, err := store.Load(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), aggID),
	)
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

	_, err := store.Load(context.Background(), id.NewStreamRef("User", id.NewStreamID()))
	if err == nil {
		t.Fatal("expected error for row scan failure")
	}
}

func TestScanEvents_InvalidMetadata(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewStreamID()
	eventID := id.NewEventID()

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(
			sqlmock.NewRows(eventColumns()).
				AddRow(
					eventID.String(),
					"UserCreated", "User", aggID.String(), 1, 1, nil, "", []byte(`{invalid`), time.Now(),
				),
		)

	_, err := store.Load(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), aggID),
	)
	if err == nil {
		t.Fatal("expected error for invalid metadata JSON")
	}
}

func TestScanEvents_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewStreamID()
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

	metaJSON, err := sqlpkg.MarshalMetadata(evt.Metadata())
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(loadQuery)).
		WithArgs("User", aggID).
		WillReturnRows(
			sqlmock.NewRows(eventColumns()).
				AddRow(
					eventID.String(),
					"UserCreated", "User", aggID.String(), 1, 1, []byte(`{"name":"test"}`), "", metaJSON, ts,
				),
		)

	loaded, err := store.Load(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), aggID),
	)
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
