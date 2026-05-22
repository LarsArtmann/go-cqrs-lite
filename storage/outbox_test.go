package storage

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newTestOutbox(t *testing.T) (*SQLOutbox, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	outbox, err := NewSQLOutbox(db)
	if err != nil {
		t.Fatalf("NewSQLOutbox: %v", err)
	}

	return outbox, mock
}

func TestNewSQLOutbox_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSQLOutbox(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestSQLOutbox_Close(t *testing.T) {
	t.Parallel()

	outbox, _ := newTestOutbox(t)

	if err := outbox.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestSQLOutbox_Append(t *testing.T) {
	t.Parallel()

	outbox, mock := newTestOutbox(t)

	aggID := id.NewAggregateID()
	evt := newTestEvent(t, "UserCreated", aggID, 1)

	events := []event.Event{evt}

	expectedJSON, err := marshalOutboxEvents(events)
	if err != nil {
		t.Fatalf("marshal events: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (id, status, events, created_at) VALUES ($1, $2, $3, $4)`)).
		WithArgs(evt.ID(), string(OutboxStatusPending), expectedJSON, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = outbox.Append(t.Context(), events)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSQLOutbox_Append_Empty(t *testing.T) {
	t.Parallel()

	outbox, _ := newTestOutbox(t)

	err := outbox.Append(t.Context(), nil)
	if err != nil {
		t.Fatalf("Append empty: %v", err)
	}
}

func TestSQLOutbox_Append_InsertError(t *testing.T) {
	t.Parallel()

	outbox, mock := newTestOutbox(t)

	aggID := id.NewAggregateID()
	evt := newTestEvent(t, "UserCreated", aggID, 1)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (id, status, events, created_at) VALUES ($1, $2, $3, $4)`)).
		WithArgs(evt.ID(), string(OutboxStatusPending), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errTestDB)

	err := outbox.Append(t.Context(), []event.Event{evt})
	if err == nil {
		t.Fatal("expected insert error")
	}
}

func TestSQLOutbox_PollPending(t *testing.T) {
	t.Parallel()

	outbox, mock := newTestOutbox(t)

	aggID := id.NewAggregateID()
	evt := newTestEvent(t, "UserCreated", aggID, 1)
	events := []event.Event{evt}

	eventsJSON, err := marshalOutboxEvents(events)
	if err != nil {
		t.Fatalf("marshal events: %v", err)
	}

	pollQuery := `SELECT id, events FROM outbox
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2`

	rows := sqlmock.NewRows([]string{"id", "events"}).
		AddRow(evt.ID().String(), eventsJSON)

	mock.ExpectQuery(regexp.QuoteMeta(pollQuery)).
		WithArgs(string(OutboxStatusPending), 10).
		WillReturnRows(rows)

	entries, err := outbox.PollPending(t.Context(), 10)
	if err != nil {
		t.Fatalf("PollPending: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].ID != event.OutboxID(evt.ID().String()) {
		t.Fatalf("expected outbox ID %s, got %s", evt.ID(), entries[0].ID)
	}

	if len(entries[0].Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(entries[0].Events))
	}

	if entries[0].Events[0].Type() != evt.Type() {
		t.Fatalf("expected event type %s, got %s", evt.Type(), entries[0].Events[0].Type())
	}
}

func TestSQLOutbox_PollPending_Empty(t *testing.T) {
	t.Parallel()

	outbox, mock := newTestOutbox(t)

	pollQuery := `SELECT id, events FROM outbox
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2`

	mock.ExpectQuery(regexp.QuoteMeta(pollQuery)).
		WithArgs(string(OutboxStatusPending), 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "events"}))

	entries, err := outbox.PollPending(t.Context(), 10)
	if err != nil {
		t.Fatalf("PollPending: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestSQLOutbox_PollPending_QueryError(t *testing.T) {
	t.Parallel()

	outbox, _ := newTestOutbox(t)

	_, err := outbox.PollPending(t.Context(), 10)
	if err == nil {
		t.Fatal("expected query error")
	}
}

func TestSQLOutbox_Ack(t *testing.T) {
	t.Parallel()

	outbox, mock := newTestOutbox(t)

	ids := []event.OutboxID{"outbox-1", "outbox-2"}

	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM outbox WHERE id IN ($1, $2)",
	)).WithArgs("outbox-1", "outbox-2").
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := outbox.Ack(t.Context(), ids)
	if err != nil {
		t.Fatalf("Ack: %v", err)
	}
}

func TestSQLOutbox_Ack_Empty(t *testing.T) {
	t.Parallel()

	outbox, _ := newTestOutbox(t)

	err := outbox.Ack(t.Context(), nil)
	if err != nil {
		t.Fatalf("Ack empty: %v", err)
	}
}

func TestSQLOutbox_Ack_DeleteError(t *testing.T) {
	t.Parallel()

	outbox, mock := newTestOutbox(t)

	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM outbox WHERE id IN ($1)",
	)).WithArgs("outbox-1").
		WillReturnError(errTestDB)

	err := outbox.Ack(t.Context(), []event.OutboxID{"outbox-1"})
	if err == nil {
		t.Fatal("expected delete error")
	}
}

func TestMarshalUnmarshalOutboxEvents_RoundTrip(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt := newTestEvent(t, "UserCreated", aggID, 1)

	data, err := marshalOutboxEvents([]event.Event{evt})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("expected valid JSON")
	}

	events, err := unmarshalOutboxEvents(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type() != evt.Type() {
		t.Fatalf("expected type %s, got %s", evt.Type(), events[0].Type())
	}

	if events[0].AggregateID() != evt.AggregateID() {
		t.Fatalf("expected aggID %s, got %s", evt.AggregateID(), events[0].AggregateID())
	}

	if events[0].Version() != evt.Version() {
		t.Fatalf("expected version %d, got %d", evt.Version(), events[0].Version())
	}
}

func newTestEvent(
	t *testing.T,
	eventType string,
	aggID id.AggregateID,
	version event.Version,
) *event.Core {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type(eventType),
		aggID,
		"User",
		version,
		[]byte(`{"name":"test"}`),
		event.WithOccurredAt(time.Now().Truncate(time.Microsecond)),
	)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	core := evt

	return core
}

func TestOutboxSchema(t *testing.T) {
	t.Parallel()

	schema := OutboxSchema()
	if schema == "" {
		t.Fatal("OutboxSchema returned empty string")
	}

	if !regexp.MustCompile(`(?i)CREATE TABLE`).MatchString(schema) {
		t.Fatal("OutboxSchema should contain CREATE TABLE")
	}
}

func TestUnmarshalOutboxEvents_InvalidAggregateID(t *testing.T) {
	t.Parallel()

	data := `[{"id":"01HXXXX","type":"UserCreated","aggregate_type":"User","aggregate_id":"invalid","version":1,"payload":null,"occurred_at":"2024-01-01T00:00:00Z"}]`

	_, err := unmarshalOutboxEvents([]byte(data))
	if err == nil {
		t.Fatal("expected error for invalid aggregate ID in outbox event")
	}
}

var errTestDB = errors.New("test db error")
