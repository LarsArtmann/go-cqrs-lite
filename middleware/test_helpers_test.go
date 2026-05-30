package middleware

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/query"
)

type testCommand struct {
	aggregateID id.AggregateID
}

func (c *testCommand) Type() command.Type          { return "test.cmd" }
func (c *testCommand) AggregateID() id.AggregateID { return c.aggregateID }

type testQuery struct{}

func (q *testQuery) Type() query.Type { return "test.query" }

type countingHandler struct {
	mu     sync.Mutex
	infos  int
	errors int
}

func newCountingHandler() *countingHandler {
	return &countingHandler{}
}

func (h *countingHandler) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

func (h *countingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if r.Level >= slog.LevelError {
		h.errors++
	} else {
		h.infos++
	}

	return nil
}

func (h *countingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *countingHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *countingHandler) InfoCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.infos
}

func (h *countingHandler) ErrorCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.errors
}

func newTestLogger() (*slog.Logger, *countingHandler) {
	h := newCountingHandler()

	return slog.New(h), h
}

func NoopCommandHandler() func(context.Context, command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}

func failingCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return errors.New(msg)
	}
}

func panicCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		panic(msg)
	}
}

func callbackCommandHandler(called *bool) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*called = true

		return nil
	}
}

func noopQueryHandler() func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, nil
	}
}

func failingQueryHandler(msg string) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New(msg)
	}
}

func panicQueryHandler(msg string) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		panic(msg)
	}
}

func callbackQueryHandler(called *bool) func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		*called = true

		return nil, nil
	}
}
