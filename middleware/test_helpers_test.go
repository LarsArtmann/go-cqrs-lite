package middleware

import (
	"context"
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
