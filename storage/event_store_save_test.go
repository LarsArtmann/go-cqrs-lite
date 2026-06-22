package storage

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestSQLEventStore_Save_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	expectSaveSuccess(mock, evt)

	err := saveEvt(t, store, evt)
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

	err := saveEvt(t, store, evt)
	if err == nil {
		t.Fatal("expected concurrency conflict error")
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Errorf("error should wrap event.ErrVersionConflict, got: %v", err)
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Errorf(
			"error should wrap event.ErrVersionConflict (ErrConcurrencyConflict), got: %v",
			err,
		)
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

	err := store.Save(
		context.Background(),
		event.NewAggregateRef("User", aggID),
		nil,
		event.Version(0),
	)
	if err != nil {
		t.Fatalf("Save with empty events: %v", err)
	}
}

func TestSQLEventStore_Save_BeginTxFailure(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := saveEvt(t, store, evt)
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

	err := saveEvt(t, store, evt)
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
	expectInsertExec(mock, evt).WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()
	err := saveEvt(t, store, evt)
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
	expectInsertSuccess(mock, evt)
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	err := saveEvt(t, store, evt)
	if err == nil {
		t.Fatal("expected error for commit failure")
	}
}

func TestSQLEventStore_AppendBatch_Success(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	aggID := id.NewAggregateID()
	evt1 := testEventWithAggID(t, "UserCreated", aggID, 1)
	evt2 := testEventWithAggID(t, "UserCreated", aggID, 2)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO events.*VALUES.*").WithArgs(
		evt1.ID(),
		"UserCreated", "User", evt1.AggregateID(), 1, evt1.SchemaVersion().Int(), evt1.Payload(), sqlmock.AnyArg(), sqlmock.AnyArg(), evt1.OccurredAt(),
		evt2.ID(),
		"UserCreated", "User", evt2.AggregateID(), 2, evt2.SchemaVersion().Int(), evt2.Payload(), sqlmock.AnyArg(), sqlmock.AnyArg(), evt2.OccurredAt(),
	).
		WillReturnResult(sqlmock.NewResult(2, 2))
	mock.ExpectCommit()

	err := store.AppendBatch(
		context.Background(),
		event.NewAggregateRef("User", evt1.AggregateID()),
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

	err := store.AppendBatch(context.Background(), event.NewAggregateRef("User", aggID), nil)
	if err != nil {
		t.Fatalf("AppendBatch with empty events: %v", err)
	}
}

func TestSQLEventStore_AppendBatch_BeginTxFailure(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := appendBatchEvt(t, store, evt)
	if err == nil {
		t.Fatal("expected error for BeginTx failure")
	}
}

func TestSQLEventStore_AppendBatch_InsertError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO events.*VALUES.*").
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()
	err := appendBatchEvt(t, store, evt)
	if err == nil {
		t.Fatal("expected error for insert failure")
	}
}

func TestSQLEventStore_AppendBatch_CommitError(t *testing.T) {
	t.Parallel()

	store, mock := newTestStore(t)
	evt := testEvent(t)

	mock.ExpectBegin()
	expectInsertSuccess(mock, evt)
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	err := appendBatchEvt(t, store, evt)
	if err == nil {
		t.Fatal("expected error for commit failure")
	}
}
