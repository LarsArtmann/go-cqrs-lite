package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestCommandValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ command.Command) error { return nil }
	mw := CommandValidation(validate)

	called := false
	handler := mw(testhelpers.CallbackCommandHandler(&called))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

func TestCommandValidation_Fail(t *testing.T) {
	t.Parallel()

	validate := func(_ command.Command) error {
		return errors.New("invalid")
	}
	mw := CommandValidation(validate)

	handler := mw(testhelpers.FailingCommandHandler("should not be called"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEventValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ event.Event) error { return nil }
	mw := EventValidation(validate)

	called := false
	handler := mw(testhelpers.CallbackEventHandler(&called))

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

func TestEventValidation_Fail(t *testing.T) {
	t.Parallel()

	validate := func(_ event.Event) error {
		return errors.New("invalid event")
	}
	mw := EventValidation(validate)

	handler := mw(testhelpers.FailingEventHandler("should not be called"))

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestQueryValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ query.Query) error { return nil }
	mw := QueryValidation(validate)

	called := false
	handler := mw(testhelpers.CallbackQueryHandler(&called))

	_, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

func TestQueryValidation_Fail(t *testing.T) {
	t.Parallel()

	validate := func(_ query.Query) error {
		return errors.New("invalid")
	}
	mw := QueryValidation(validate)

	handler := mw(testhelpers.FailingQueryHandler("should not be called"))

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCommandValidation_SentinelError(t *testing.T) {
	t.Parallel()

	mw := CommandValidation(func(_ command.Command) error {
		return errors.New("invalid")
	})
	handler := mw(testhelpers.NoopCommandHandler())

	err := handler(context.Background(), &testCommand{aggregateID: id.NewAggregateID()})
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected errors.Is(err, ErrValidationFailed), got %v", err)
	}
}

func TestEventValidation_SentinelError(t *testing.T) {
	t.Parallel()

	mw := EventValidation(func(_ event.Event) error {
		return errors.New("invalid")
	})
	handler := mw(testhelpers.NoopEventHandler())

	evt, evtErr := testhelpers.NewTestEvent()
	if evtErr != nil {
		t.Fatalf("NewTestEvent: %v", evtErr)
	}

	err := handler(context.Background(), evt)
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected errors.Is(err, ErrValidationFailed), got %v", err)
	}
}
