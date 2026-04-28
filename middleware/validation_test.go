package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestCommandValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ any) error { return nil }
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

	validate := func(_ any) error {
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

	validate := func(_ any) error { return nil }
	mw := EventValidation(validate)

	called := false
	handler := mw(func(_ context.Context, _ event.Event) error {
		called = true

		return nil
	})

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
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

	validate := func(_ any) error {
		return errors.New("invalid event")
	}
	mw := EventValidation(validate)

	handler := mw(testhelpers.FailingEventHandler("should not be called"))

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
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

	validate := func(_ any) error { return nil }
	mw := QueryValidation(validate)

	called := false
	handler := mw(func(_ context.Context, q query.Query) (any, error) {
		called = true

		return q.Type(), nil
	})

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

	validate := func(_ any) error {
		return errors.New("invalid")
	}
	mw := QueryValidation(validate)

	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return "should not be called", nil
	})

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
