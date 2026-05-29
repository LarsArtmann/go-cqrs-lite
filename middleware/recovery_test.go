package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestCommandRecovery_NoPanic(t *testing.T) {
	t.Parallel()

	mw := CommandRecovery()
	handler := mw(testhelpers.NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandRecovery_Panic(t *testing.T) {
	t.Parallel()

	mw := CommandRecovery()
	handler := mw(testhelpers.PanicCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}

	testhelpers.AssertErrorContains(t, err, "panic recovered in command test.cmd: boom")
}

func TestEventRecovery_NoPanic(t *testing.T) {
	t.Parallel()

	mw := EventRecovery()
	handler := mw(testhelpers.NoopEventHandler())

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventRecovery_Panic(t *testing.T) {
	t.Parallel()

	mw := EventRecovery()
	handler := mw(testhelpers.PanicEventHandler("event boom"))

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}

	testhelpers.AssertErrorContains(t, err, "panic recovered in event test.evt: event boom")
}

func TestQueryRecovery_NoPanic(t *testing.T) {
	t.Parallel()

	mw := QueryRecovery()
	handler := mw(testhelpers.NoopQueryHandler())

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestQueryRecovery_Panic(t *testing.T) {
	t.Parallel()

	mw := QueryRecovery()
	handler := mw(testhelpers.PanicQueryHandler("query boom"))

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}

	testhelpers.AssertErrorContains(t, err, "panic recovered in query test.query: query boom")
}

func TestCommandRecovery_SentinelError(t *testing.T) {
	t.Parallel()

	mw := CommandRecovery()
	handler := mw(testhelpers.PanicCommandHandler("boom"))

	err := handler(context.Background(), &testCommand{aggregateID: id.NewAggregateID()})
	if !errors.Is(err, ErrPanicRecovered) {
		t.Errorf("expected errors.Is(err, ErrPanicRecovered), got %v", err)
	}
}
