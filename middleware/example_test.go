package middleware_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
)

func ExampleNewRecovery() {
	adapter := middleware.EventAdapter

	mw := middleware.NewRecovery(adapter)

	var called bool

	handler := func(_ context.Context, _ event.Event) error {
		called = true
		panic("something went wrong")
	}

	wrapped := mw(handler)

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("UserCreated", aggID, "User", 1, []byte(`{}`))

	err := wrapped(context.Background(), evt)
	fmt.Println(called, err != nil)

	// Output:
	// true true
}

func ExampleNewLogging() {
	adapter := middleware.EventAdapter
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	mw := middleware.NewLogging(adapter, logger)
	fmt.Println(mw != nil)

	// Output:
	// true
}
