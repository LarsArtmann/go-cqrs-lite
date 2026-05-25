package storage

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func newTestTransactionalStore(t *testing.T) (*SQLTransactionalStore, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	store, err := NewSQLEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}

	outbox, err := NewSQLOutbox(db)
	if err != nil {
		t.Fatalf("NewSQLOutbox: %v", err)
	}

	ts, err := NewSQLTransactionalStore(store, outbox)
	if err != nil {
		t.Fatalf("NewSQLTransactionalStore: %v", err)
	}

	return ts, mock
}

func saveWithOutboxEvt(t *testing.T, ts *SQLTransactionalStore, evt *event.ImmutableEvent) error {
	t.Helper()

	return ts.SaveWithOutbox(
		context.Background(),
		"User",
		evt.AggregateID(),
		[]event.Event{evt},
		event.Version(0),
	)
}

func expectOutboxInsertSuccess(mock sqlmock.Sqlmock, evt *event.ImmutableEvent) {
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (id, status, events, created_at) VALUES ($1, $2, $3, $4)`)).
		WithArgs(
			evt.ID(), string(OutboxStatusPending), sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func expectOutboxInsertError(mock sqlmock.Sqlmock, evt *event.ImmutableEvent, err error) {
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO outbox (id, status, events, created_at) VALUES ($1, $2, $3, $4)`)).
		WithArgs(
			evt.ID(), string(OutboxStatusPending), sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(err)
}

func TestNewSQLTransactionalStore_NilStore(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	outbox, err := NewSQLOutbox(db)
	if err != nil {
		t.Fatalf("NewSQLOutbox: %v", err)
	}

	_, err = NewSQLTransactionalStore(nil, outbox)
	if err == nil {
		t.Fatal("expected error for nil store")
	}

	if !errors.Is(err, ErrNilDB) {
		t.Errorf("error should wrap ErrNilDB, got: %v", err)
	}
}

func TestNewSQLTransactionalStore_NilOutbox(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	store, err := NewSQLEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}

	_, err = NewSQLTransactionalStore(store, nil)
	if err == nil {
		t.Fatal("expected error for nil outbox")
	}

	if !errors.Is(err, ErrNilDB) {
		t.Errorf("error should wrap ErrNilDB, got: %v", err)
	}
}

func TestSQLTransactionalStore_SaveWithOutbox_Success(t *testing.T) {
	t.Parallel()

	ts, mock := newTestTransactionalStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 0)
	expectInsertSuccess(mock, evt)
	expectOutboxInsertSuccess(mock, evt)
	mock.ExpectCommit()

	err := saveWithOutboxEvt(t, ts, evt)
	if err != nil {
		t.Fatalf("SaveWithOutbox: %v", err)
	}

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestSQLTransactionalStore_SaveWithOutbox_EmptyEvents(t *testing.T) {
	t.Parallel()

	ts, _ := newTestTransactionalStore(t)

	err := ts.SaveWithOutbox(
		context.Background(),
		"User",
		id.NewAggregateID(),
		nil,
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("SaveWithOutbox with empty events: %v", err)
	}
}

func TestSQLTransactionalStore_SaveWithOutbox_BeginTxFailure(t *testing.T) {
	t.Parallel()

	ts, mock := newTestTransactionalStore(t)
	evt := testEvent(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := saveWithOutboxEvt(t, ts, evt)
	if err == nil {
		t.Fatal("expected error for BeginTx failure")
	}
}

func TestSQLTransactionalStore_SaveWithOutbox_VersionConflict(t *testing.T) {
	t.Parallel()

	ts, mock := newTestTransactionalStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 5)
	mock.ExpectRollback()

	err := saveWithOutboxEvt(t, ts, evt)
	if err == nil {
		t.Fatal("expected version conflict error")
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Errorf("error should wrap ErrVersionConflict, got: %v", err)
	}
}

func TestSQLTransactionalStore_SaveWithOutbox_InsertEventFailure(t *testing.T) {
	t.Parallel()

	ts, mock := newTestTransactionalStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 0)
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).WithArgs(
		evt.ID(),
		"UserCreated", "User", evt.AggregateID(), 1, 1, evt.Payload(), sqlmock.AnyArg(), evt.OccurredAt(),
	).
		WillReturnError(errors.New("insert event failed"))
	mock.ExpectRollback()

	err := saveWithOutboxEvt(t, ts, evt)
	if err == nil {
		t.Fatal("expected error for event insert failure")
	}
}

func TestSQLTransactionalStore_SaveWithOutbox_OutboxInsertFailure(t *testing.T) {
	t.Parallel()

	ts, mock := newTestTransactionalStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 0)
	expectInsertSuccess(mock, evt)
	expectOutboxInsertError(mock, evt, errors.New("outbox insert failed"))
	mock.ExpectRollback()

	err := saveWithOutboxEvt(t, ts, evt)
	if err == nil {
		t.Fatal("expected error for outbox insert failure")
	}
}

func TestSQLTransactionalStore_SaveWithOutbox_CommitFailure(t *testing.T) {
	t.Parallel()

	ts, mock := newTestTransactionalStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectVersionCheck(mock, evt.AggregateID(), 0)
	expectInsertSuccess(mock, evt)
	expectOutboxInsertSuccess(mock, evt)
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err := saveWithOutboxEvt(t, ts, evt)
	if err == nil {
		t.Fatal("expected error for commit failure")
	}
}
