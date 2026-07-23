package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestSQLEventStore_SaveMultiBatch_Success(t *testing.T) {
	store, mock := newTestStore(t)
	ctx := context.Background()

	aggA := id.NewStreamID()
	aggB := id.NewStreamID()

	evtA := testEventWithAggID(t, "UserCreated", aggA, 1)
	evtB := testEventWithAggID(t, "UserCreated", aggB, 1)

	entries := []event.MultiBatchEntry{
		{Ref: id.NewStreamRef("User", aggA), Events: []event.Event{evtA}},
		{Ref: id.NewStreamRef("User", aggB), Events: []event.Event{evtB}},
	}

	mock.ExpectBegin()
	expectInsertSuccess(mock, evtA)
	expectInsertSuccess(mock, evtB)
	mock.ExpectCommit()

	err := store.SaveMultiBatch(ctx, entries)
	if err != nil {
		t.Fatalf("SaveMultiBatch failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestSQLEventStore_SaveMultiBatch_Empty(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	err := store.SaveMultiBatch(ctx, nil)
	if err != nil {
		t.Fatalf("SaveMultiBatch with nil should succeed: %v", err)
	}

	err = store.SaveMultiBatch(ctx, []event.MultiBatchEntry{})
	if err != nil {
		t.Fatalf("SaveMultiBatch with empty slice should succeed: %v", err)
	}
}

func TestSQLEventStore_SaveMultiBatch_BeginTxFailure(t *testing.T) {
	store, mock := newTestStore(t)
	ctx := context.Background()

	aggA := id.NewStreamID()
	evtA := testEventWithAggID(t, "UserCreated", aggA, 1)

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := store.SaveMultiBatch(ctx, []event.MultiBatchEntry{
		{Ref: id.NewStreamRef("User", aggA), Events: []event.Event{evtA}},
	})
	if err == nil {
		t.Fatal("expected error from BeginTx failure")
	}
}

func TestSQLEventStore_SaveMultiBatch_InsertError_RollsBack(t *testing.T) {
	store, mock := newTestStore(t)
	ctx := context.Background()

	aggA := id.NewStreamID()
	aggB := id.NewStreamID()

	evtA := testEventWithAggID(t, "UserCreated", aggA, 1)
	evtB := testEventWithAggID(t, "UserCreated", aggB, 1)

	entries := []event.MultiBatchEntry{
		{Ref: id.NewStreamRef("User", aggA), Events: []event.Event{evtA}},
		{Ref: id.NewStreamRef("User", aggB), Events: []event.Event{evtB}},
	}

	mock.ExpectBegin()
	expectInsertSuccess(mock, evtA)
	expectInsertExec(mock, evtB).WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	err := store.SaveMultiBatch(ctx, entries)
	if err == nil {
		t.Fatal("expected error from insert failure")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestSQLEventStore_SaveMultiBatch_CommitError(t *testing.T) {
	store, mock := newTestStore(t)
	ctx := context.Background()

	aggA := id.NewStreamID()
	evtA := testEventWithAggID(t, "UserCreated", aggA, 1)

	mock.ExpectBegin()
	expectInsertSuccess(mock, evtA)
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err := store.SaveMultiBatch(ctx, []event.MultiBatchEntry{
		{Ref: id.NewStreamRef("User", aggA), Events: []event.Event{evtA}},
	})
	if err == nil {
		t.Fatal("expected error from commit failure")
	}
}

func TestSQLEventStore_SaveMultiBatch_SkipsEmptyEntries(t *testing.T) {
	store, mock := newTestStore(t)
	ctx := context.Background()

	aggA := id.NewStreamID()
	evtA := testEventWithAggID(t, "UserCreated", aggA, 1)

	entries := []event.MultiBatchEntry{
		{Ref: id.NewStreamRef("User", aggA), Events: []event.Event{evtA}},
		{Ref: id.NewStreamRef("User", id.NewStreamID()), Events: nil},
	}

	mock.ExpectBegin()
	expectInsertSuccess(mock, evtA)
	mock.ExpectCommit()

	err := store.SaveMultiBatch(ctx, entries)
	if err != nil {
		t.Fatalf("SaveMultiBatch failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestSQLEventStore_MultiSinkInterface(t *testing.T) {
	store, _ := newTestStore(t)

	var _ event.MultiSink = store
}
